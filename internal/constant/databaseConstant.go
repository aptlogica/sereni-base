// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constant

type Function struct {
	FunctionName   string
	FunctionParams string
	FunctionSQL    string
}

var DefinedFunctions = []Function{
	{
		FunctionName:   "dynamic_array_join_assets_jsonb",
		FunctionParams: "schema_name TEXT, source_table TEXT, source_columns TEXT[], target_table TEXT",
		FunctionSQL: `
			RETURNS SETOF JSONB
			LANGUAGE plpgsql AS $$
			DECLARE
				query TEXT;
				col_exprs TEXT;
			BEGIN
				-- Build JSONB expressions for each source column
				col_exprs := array_to_string(
					ARRAY(
						SELECT format(
							'%L, COALESCE((SELECT jsonb_agg(a) FROM %I.%I a WHERE a.id = ANY(p.%I::uuid[])), ''[]''::jsonb)',
							c, schema_name, target_table, c
						)
						FROM unnest(source_columns) AS c
					),
					', '
				);

				-- Final query with ORDER BY p.id
				query := format(
					'SELECT to_jsonb(p) || jsonb_build_object(%s) AS result FROM %I.%I p ORDER BY p.id',
					col_exprs,
					schema_name,
					source_table
				);

				RETURN QUERY EXECUTE query;
			END;
			$$;
		`,
	},
	{
		FunctionName:   "convert_column_type",
		FunctionParams: "schema_name TEXT, table_name TEXT, column_name TEXT, target_type TEXT, empty_before BOOLEAN",
		FunctionSQL: `
			RETURNS void
			LANGUAGE plpgsql
			AS $$
			DECLARE
				full_table_name TEXT;
				sql TEXT;
			BEGIN
				-- Build full table name
				full_table_name := format('%I.%I', schema_name, table_name);

				-- If user wants to empty before converting
				IF empty_before THEN
					sql := format(
						'ALTER TABLE %s ALTER COLUMN %I TYPE %s USING NULL',
						full_table_name,
						column_name,
						target_type
					);
					EXECUTE sql;
					RETURN;
				END IF;

				-- Normal behavior: try conversion with CAST → fallback to NULL
				sql := format(
					'ALTER TABLE %s ALTER COLUMN %I TYPE %s USING CAST(CAST(%I AS text) AS %s)',
					full_table_name,
					column_name,
					target_type,
					column_name,
					target_type
				);

				BEGIN
					EXECUTE sql;
				EXCEPTION WHEN OTHERS THEN
					-- Fallback: clear data if cast fails
					sql := format(
						'ALTER TABLE %s ALTER COLUMN %I TYPE %s USING NULL',
						full_table_name,
						column_name,
						target_type
					);
					EXECUTE sql;
				END;
			END;
			$$;
		`,
	},
	{
		FunctionName:   "get_table_data_with_relation",
		FunctionParams: "schema_name TEXT, source_table_name TEXT, relation_data JSON[]",
		FunctionSQL: `
			RETURNS JSON
			LANGUAGE plpgsql AS
			$$
			DECLARE
				rel JSON;
				source_column_name TEXT;
				target_table_name TEXT;
				target_column_name TEXT;
				relation TEXT;
				target_columns TEXT[];
				query TEXT;
				relation_sql TEXT := '';
				cols TEXT;
				result JSON;
				is_array BOOLEAN;
			BEGIN
				-- Loop through each relation object in relation_data array
				FOR rel IN SELECT * FROM unnest(relation_data)
				LOOP
					source_column_name := rel->>'source_column_name';
					target_table_name  := rel->>'target_table_name';
					target_column_name := rel->>'target_column_name';
					relation           := rel->>'relation';
					target_columns     := ARRAY(SELECT json_array_elements_text(rel->'target_columns'));

					-- Detect if the source column is an array type
					SELECT (data_type = 'ARRAY') INTO is_array
					FROM information_schema.columns
					WHERE table_schema = schema_name
					  AND table_name = source_table_name
					  AND column_name = source_column_name
					LIMIT 1;

					-- Build columns for this relation (handle scalar vs array source columns)
					cols := array_to_string(
						ARRAY(
							SELECT CASE
								WHEN relation IN ('has-many','many-to-many') THEN
									CASE WHEN is_array THEN
										format(
											'(SELECT COALESCE(JSON_AGG(t.%I), ''[]''::JSON)
											  FROM %I.%I t
											  WHERE t.%I = ANY(s.%I)) AS %I',
											c, schema_name, target_table_name, target_column_name, source_column_name,
											target_table_name || '_' || c
										)
									ELSE
										format(
											'(SELECT COALESCE(JSON_AGG(t.%I), ''[]''::JSON)
											  FROM %I.%I t
											  WHERE t.%I = s.%I) AS %I',
											c, schema_name, target_table_name, target_column_name, source_column_name,
											target_table_name || '_' || c
										)
									END
								ELSE
									format(
										'(SELECT t.%I FROM %I.%I t WHERE t.%I = s.%I LIMIT 1) AS %I',
										c, schema_name, target_table_name, target_column_name, source_column_name,
										target_table_name || '_' || c
									)
								END
							FROM unnest(target_columns) AS c
						), ', '
					);

					-- Add this relation's columns to the main query projection
					relation_sql := relation_sql || ', ' || cols;
				END LOOP;

				-- Build final query without pagination
				query := format(
					'SELECT COALESCE(JSON_AGG(row_to_json(row)), ''[]''::JSON)
					 FROM (
						 SELECT s.* %s
						 FROM %I.%I s
						 ORDER BY s.created_time
					 ) row',
					relation_sql,
					schema_name, source_table_name
				);

				-- Execute dynamic query into a JSON array
				EXECUTE query INTO result;

				RETURN result;
			END;
			$$;
		`,
	},
	// --- Link functions --------------------------------------------------------------------------
	// Link cells hold a JSONB array of the linked rows' primary-key values, e.g.
	// [{"org_code": "IN", "dept_code": "HR"}]; a default table's key is {"id": 12}. Callers still pass
	// row ids; the functions read each row's key from its primary key constraint. p_table values are
	// already-quoted "schema"."table" names taken from metadata; column names are quoted here with %I.
	{
		// Primary-key columns of a table, in key order (falls back to id).
		FunctionName:   "link_key_columns",
		FunctionParams: "p_table TEXT",
		FunctionSQL: `
			RETURNS TEXT[]
			LANGUAGE sql STABLE AS
			$$
				SELECT COALESCE(
					(SELECT array_agg(a.attname::text ORDER BY k.ord)
					 FROM pg_constraint c
					 CROSS JOIN LATERAL unnest(c.conkey) WITH ORDINALITY AS k(attnum, ord)
					 JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = k.attnum
					 WHERE c.conrelid = to_regclass(p_table) AND c.contype = 'p'),
					ARRAY['id']);
			$$;
		`,
	},
	{
		// SQL expression that builds the key object of the row aliased p_alias.
		FunctionName:   "link_key_expr",
		FunctionParams: "p_table TEXT, p_alias TEXT",
		FunctionSQL: `
			RETURNS TEXT
			LANGUAGE sql STABLE AS
			$$
				SELECT 'jsonb_build_object(' || string_agg(format('%L, %I.%I', c, p_alias, c), ', ') || ')'
				FROM unnest(public.link_key_columns(p_table)) AS c;
			$$;
		`,
	},
	{
		// SQL condition joining two aliases of the same table on its key columns.
		FunctionName:   "link_key_join",
		FunctionParams: "p_table TEXT, p_left TEXT, p_right TEXT",
		FunctionSQL: `
			RETURNS TEXT
			LANGUAGE sql STABLE AS
			$$
				SELECT string_agg(format('%I.%I = %I.%I', p_left, c, p_right, c), ' AND ')
				FROM unnest(public.link_key_columns(p_table)) AS c;
			$$;
		`,
	},
	{
		// Key object of one row.
		FunctionName:   "link_row_key",
		FunctionParams: "p_table TEXT, p_row BIGINT",
		FunctionSQL: `
			RETURNS JSONB
			LANGUAGE plpgsql STABLE AS
			$$
			DECLARE
				row_key JSONB;
			BEGIN
				EXECUTE format('SELECT %s FROM %s t WHERE t.id = $1', public.link_key_expr(p_table, 't'), p_table)
					INTO row_key USING p_row;
				RETURN row_key;
			END;
			$$;
		`,
	},
	{
		// Locks a row for the rest of the calling statement and reports whether it exists.
		FunctionName:   "link_lock_row",
		FunctionParams: "p_table TEXT, p_row BIGINT",
		FunctionSQL: `
			RETURNS BOOLEAN
			LANGUAGE plpgsql AS
			$$
			DECLARE
				locked_id BIGINT;
			BEGIN
				EXECUTE format('SELECT id::bigint FROM %s WHERE id = $1 FOR UPDATE', p_table)
					INTO locked_id USING p_row;
				RETURN locked_id IS NOT NULL;
			END;
			$$;
		`,
	},
	{
		// Adds (p_link) or removes p_key in one row's link column. Single-valued sides hold at most one key.
		FunctionName:   "link_apply_change",
		FunctionParams: "p_table TEXT, p_column TEXT, p_is_array BOOLEAN, p_row BIGINT, p_key JSONB, p_link BOOLEAN, p_updated_by TEXT",
		FunctionSQL: `
			RETURNS VOID
			LANGUAGE plpgsql AS
			$$
			DECLARE
				set_expr TEXT;
				where_expr TEXT := 'id = $2';
				audit TEXT := ', last_modified_time = $3';
			BEGIN
				IF p_is_array AND p_link THEN
					set_expr := format(
						'%1$I = CASE WHEN COALESCE(%1$I, ''[]''::jsonb) @> jsonb_build_array($1) THEN %1$I ELSE COALESCE(%1$I, ''[]''::jsonb) || jsonb_build_array($1) END',
						p_column);
				ELSIF p_link THEN
					set_expr := format('%I = jsonb_build_array($1)', p_column);
				ELSE
					set_expr := format(
						'%1$I = (SELECT jsonb_agg(x.e ORDER BY x.o) FROM jsonb_array_elements(%1$I) WITH ORDINALITY AS x(e, o) WHERE x.e <> $1)',
						p_column);
					where_expr := format('id = $2 AND %I @> jsonb_build_array($1)', p_column);
				END IF;

				IF COALESCE(p_updated_by, '') <> '' THEN
					audit := audit || ', last_modified_by = $4';
				END IF;

				EXECUTE format('UPDATE %s SET %s%s WHERE %s', p_table, set_expr, audit, where_expr)
					USING p_key, p_row, (now() AT TIME ZONE 'UTC'), p_updated_by;
			END;
			$$;
		`,
	},
	{
		// Returns (and locks) the rows whose link column holds p_key.
		FunctionName:   "link_rows_holding",
		FunctionParams: "p_table TEXT, p_column TEXT, p_key JSONB",
		FunctionSQL: `
			RETURNS SETOF BIGINT
			LANGUAGE plpgsql AS
			$$
			BEGIN
				RETURN QUERY EXECUTE format('SELECT id::bigint FROM %s WHERE %I @> jsonb_build_array($1) FOR UPDATE', p_table, p_column)
					USING p_key;
			END;
			$$;
		`,
	},
	{
		// Would making p_parent the parent of p_child create a loop? p_column points from a child to its parent.
		FunctionName:   "link_cycle_parent",
		FunctionParams: "p_table TEXT, p_column TEXT, p_parent BIGINT, p_child BIGINT",
		FunctionSQL: `
			RETURNS BOOLEAN
			LANGUAGE plpgsql AS
			$$
			DECLARE
				result BOOLEAN;
			BEGIN
				IF p_parent = p_child THEN
					RETURN TRUE;
				END IF;
				EXECUTE format(
					'WITH RECURSIVE ancestors(k) AS (
						SELECT e FROM %2$s t, jsonb_array_elements(t.%1$I) AS e WHERE t.id = $1
						UNION
						SELECT e FROM ancestors a
						CROSS JOIN LATERAL jsonb_populate_record(NULL::%2$s, a.k) AS r
						JOIN %2$s t ON %3$s
						CROSS JOIN LATERAL jsonb_array_elements(t.%1$I) AS e
					)
					SELECT EXISTS (SELECT 1 FROM ancestors WHERE k = $2)',
					p_column, p_table, public.link_key_join(p_table, 't', 'r'))
					INTO result USING p_parent, public.link_row_key(p_table, p_child);
				RETURN result;
			END;
			$$;
		`,
	},
	{
		// Would adding p_child to p_parent's children (a multi-valued column) create a loop?
		FunctionName:   "link_cycle_children",
		FunctionParams: "p_table TEXT, p_column TEXT, p_parent BIGINT, p_child BIGINT",
		FunctionSQL: `
			RETURNS BOOLEAN
			LANGUAGE plpgsql AS
			$$
			DECLARE
				result BOOLEAN;
				key_expr TEXT := public.link_key_expr(p_table, 't');
			BEGIN
				IF p_parent = p_child THEN
					RETURN TRUE;
				END IF;
				EXECUTE format(
					'WITH RECURSIVE ancestors(id, k) AS (
						SELECT t.id::bigint, %3$s FROM %2$s t WHERE t.%1$I @> jsonb_build_array($1)
						UNION
						SELECT t.id::bigint, %3$s FROM %2$s t JOIN ancestors a ON t.%1$I @> jsonb_build_array(a.k)
					)
					SELECT EXISTS (SELECT 1 FROM ancestors WHERE id = $2)',
					p_column, p_table, key_expr)
					INTO result USING public.link_row_key(p_table, p_parent), p_child;
				RETURN result;
			END;
			$$;
		`,
	},
	{
		// Links or unlinks two rows. Returns 'ok', 'source_not_found', 'target_not_found' or 'cycle';
		// every non-ok status is returned before anything is written.
		FunctionName:   "link_rows",
		FunctionParams: "p_relation_type TEXT, p_single_column BOOLEAN, p_is_self BOOLEAN, p_action TEXT, p_source_table TEXT, p_source_column TEXT, p_source_is_array BOOLEAN, p_target_table TEXT, p_target_column TEXT, p_target_is_array BOOLEAN, p_source_row BIGINT, p_target_row BIGINT, p_updated_by TEXT",
		FunctionSQL: `
			RETURNS TEXT
			LANGUAGE plpgsql AS
			$$
			DECLARE
				is_link BOOLEAN := p_action = 'link';
				holder BIGINT;
				has_cycle BOOLEAN;
				source_key JSONB;
				target_key JSONB;
			BEGIN
				-- Lock both rows in a fixed order so opposite link calls can't deadlock.
				IF p_target_table COLLATE "C" < p_source_table COLLATE "C"
					OR (p_target_table = p_source_table AND p_target_row < p_source_row) THEN
					IF NOT public.link_lock_row(p_target_table, p_target_row) THEN RETURN 'target_not_found'; END IF;
					IF NOT public.link_lock_row(p_source_table, p_source_row) THEN RETURN 'source_not_found'; END IF;
				ELSE
					IF NOT public.link_lock_row(p_source_table, p_source_row) THEN RETURN 'source_not_found'; END IF;
					IF NOT public.link_lock_row(p_target_table, p_target_row) THEN RETURN 'target_not_found'; END IF;
				END IF;

				-- Link cells store the other row's primary-key values, not its id.
				source_key := public.link_row_key(p_source_table, p_source_row);
				target_key := public.link_row_key(p_target_table, p_target_row);

				-- One-way self-link: only the edited row's cell changes.
				IF p_single_column THEN
					IF NOT is_link THEN
						PERFORM public.link_apply_change(p_source_table, p_source_column, p_source_is_array, p_source_row, target_key, FALSE, p_updated_by);
						RETURN 'ok';
					END IF;
					IF p_relation_type = 'has-many'
						AND public.link_cycle_children(p_source_table, p_source_column, p_source_row, p_target_row) THEN
						RETURN 'cycle';
					END IF;
					IF p_relation_type IN ('one-to-one', 'has-many') THEN
						FOR holder IN SELECT * FROM public.link_rows_holding(p_source_table, p_source_column, target_key) LOOP
							IF holder <> p_source_row THEN
								PERFORM public.link_apply_change(p_source_table, p_source_column, p_source_is_array, holder, target_key, FALSE, p_updated_by);
							END IF;
						END LOOP;
					END IF;
					PERFORM public.link_apply_change(p_source_table, p_source_column, p_source_is_array, p_source_row, target_key, TRUE, p_updated_by);
					RETURN 'ok';
				END IF;

				IF NOT is_link THEN
					PERFORM public.link_apply_change(p_source_table, p_source_column, p_source_is_array, p_source_row, target_key, FALSE, p_updated_by);
					PERFORM public.link_apply_change(p_target_table, p_target_column, p_target_is_array, p_target_row, source_key, FALSE, p_updated_by);
					RETURN 'ok';
				END IF;

				-- Two-column self-link has-many: the single-valued side points from a child to its parent.
				IF p_is_self AND p_relation_type = 'has-many' THEN
					IF NOT p_source_is_array THEN
						has_cycle := public.link_cycle_parent(p_source_table, p_source_column, p_target_row, p_source_row);
					ELSE
						has_cycle := public.link_cycle_parent(p_target_table, p_target_column, p_source_row, p_target_row);
					END IF;
					IF has_cycle THEN
						RETURN 'cycle';
					END IF;
				END IF;

				-- Cardinality: one-to-one partners and has-many parents release the record first.
				IF p_relation_type = 'one-to-one' OR (p_relation_type = 'has-many' AND p_source_is_array) THEN
					FOR holder IN SELECT * FROM public.link_rows_holding(p_source_table, p_source_column, target_key) LOOP
						PERFORM public.link_apply_change(p_source_table, p_source_column, p_source_is_array, holder, target_key, FALSE, p_updated_by);
						PERFORM public.link_apply_change(p_target_table, p_target_column, p_target_is_array, p_target_row, public.link_row_key(p_source_table, holder), FALSE, p_updated_by);
					END LOOP;
				END IF;
				IF p_relation_type = 'one-to-one' OR (p_relation_type = 'has-many' AND p_target_is_array) THEN
					FOR holder IN SELECT * FROM public.link_rows_holding(p_target_table, p_target_column, source_key) LOOP
						PERFORM public.link_apply_change(p_target_table, p_target_column, p_target_is_array, holder, source_key, FALSE, p_updated_by);
						PERFORM public.link_apply_change(p_source_table, p_source_column, p_source_is_array, p_source_row, public.link_row_key(p_target_table, holder), FALSE, p_updated_by);
					END LOOP;
				END IF;

				PERFORM public.link_apply_change(p_source_table, p_source_column, p_source_is_array, p_source_row, target_key, TRUE, p_updated_by);
				PERFORM public.link_apply_change(p_target_table, p_target_column, p_target_is_array, p_target_row, source_key, TRUE, p_updated_by);
				RETURN 'ok';
			END;
			$$;
		`,
	},
	{
		// Lookup values: each target's column is read from the rows whose keys the link cell holds.
		FunctionName:   "get_table_data_with_lookups",
		FunctionParams: "schema_name TEXT, source_table_name TEXT, relation_data JSON[]",
		FunctionSQL: `
			RETURNS JSON
			LANGUAGE plpgsql AS
			$$
			DECLARE
				rel JSON;
				target JSON;
				source_column_name TEXT;
				target_table TEXT;
				relation TEXT;
				is_many BOOLEAN;
				key_join TEXT;
				from_sql TEXT;
				expr TEXT;
				relation_sql TEXT := '';
				query TEXT;
				result JSON;
			BEGIN
				FOR rel IN SELECT * FROM unnest(COALESCE(relation_data, ARRAY[]::JSON[]))
				LOOP
					source_column_name := rel->>'source_column_name';
					target_table       := format('%I.%I', schema_name, rel->>'target_table_name');
					relation           := rel->>'relation';
					is_many            := COALESCE((rel->>'is_array')::BOOLEAN, FALSE) OR relation IN ('has-many', 'many-to-many');
					key_join           := public.link_key_join(target_table, 't', 'r');
					from_sql := format(
						'FROM jsonb_array_elements(s.%I) WITH ORDINALITY AS x(k, o)
						 CROSS JOIN LATERAL jsonb_populate_record(NULL::%s, x.k) AS r
						 JOIN %s t ON %s',
						source_column_name, target_table, target_table, key_join);

					FOR target IN SELECT * FROM json_array_elements(rel->'targets')
					LOOP
						IF is_many THEN
							expr := format('(SELECT COALESCE(JSON_AGG(t.%I ORDER BY x.o), ''[]''::JSON) %s) AS %I',
								target->>'column', from_sql, target->>'alias');
						ELSE
							expr := format('(SELECT t.%I %s ORDER BY x.o LIMIT 1) AS %I',
								target->>'column', from_sql, target->>'alias');
						END IF;
						relation_sql := relation_sql || ', ' || expr;
					END LOOP;
				END LOOP;

				query := format(
					'SELECT COALESCE(JSON_AGG(row_to_json(row)), ''[]''::JSON)
					 FROM (
						 SELECT s.* %s
						 FROM %I.%I s
						 ORDER BY s.created_time
					 ) row',
					relation_sql,
					schema_name, source_table_name
				);

				EXECUTE query INTO result;

				RETURN result;
			END;
			$$;
		`,
	},
	{
		// Statement trigger on every dynamic table: when rows are deleted or their key values change,
		// rewrite (or remove) the stored keys in every link column that points at this table.
		FunctionName:   "link_key_sync",
		FunctionParams: "",
		FunctionSQL: `
			RETURNS TRIGGER
			LANGUAGE plpgsql AS
			$$
			DECLARE
				this_table TEXT := format('%I.%I', TG_TABLE_SCHEMA, TG_TABLE_NAME);
				this_model TEXT;
				old_key TEXT := public.link_key_expr(this_table, 'o');
				new_key TEXT := public.link_key_expr(this_table, 'n');
				changes JSONB;
				ref RECORD;
			BEGIN
				EXECUTE format('SELECT id::text FROM %I.models WHERE alias = $1 LIMIT 1', TG_TABLE_SCHEMA)
					INTO this_model USING TG_TABLE_NAME;
				IF this_model IS NULL THEN
					RETURN NULL;
				END IF;

				IF TG_OP = 'UPDATE' THEN
					EXECUTE format(
						'SELECT jsonb_agg(jsonb_build_object(''old'', %1$s, ''new'', %2$s))
						 FROM link_old_rows o JOIN link_new_rows n ON n.id = o.id
						 WHERE %1$s IS DISTINCT FROM %2$s', old_key, new_key)
						INTO changes;
				ELSE
					EXECUTE format('SELECT jsonb_agg(jsonb_build_object(''old'', %s)) FROM link_old_rows o', old_key)
						INTO changes;
				END IF;
				IF changes IS NULL THEN
					RETURN NULL;
				END IF;

				FOR ref IN EXECUTE format(
					'SELECT m.alias, c.column_name
					 FROM %1$I.columns c JOIN %1$I.models m ON m.id::text = c.model_id::text
					 WHERE c.uidt = ''links'' AND NOT COALESCE(c.deleted, FALSE) AND c.meta->''relation''->>''with'' = $1', TG_TABLE_SCHEMA)
					USING this_model
				LOOP
					-- Matched keys are replaced by the new key on UPDATE and dropped on DELETE.
					EXECUTE format(
						'UPDATE %1$I.%2$I s SET %3$I = (
							SELECT jsonb_agg(COALESCE(ch->''new'', x.e) ORDER BY x.o)
							FROM jsonb_array_elements(s.%3$I) WITH ORDINALITY AS x(e, o)
							LEFT JOIN jsonb_array_elements($1) AS ch ON ch->''old'' = x.e
							WHERE ch IS NULL OR ch ? ''new''
						), last_modified_time = (now() AT TIME ZONE ''UTC'')
						WHERE EXISTS (SELECT 1 FROM jsonb_array_elements($1) AS ch WHERE s.%3$I @> jsonb_build_array(ch->''old''))',
						TG_TABLE_SCHEMA, ref.alias, ref.column_name)
						USING changes;
				END LOOP;
				RETURN NULL;
			END;
			$$;
		`,
	},
	{
		// Installs the key-sync triggers on one dynamic table (idempotent).
		FunctionName:   "link_install_key_sync",
		FunctionParams: "p_schema TEXT, p_table TEXT",
		FunctionSQL: `
			RETURNS VOID
			LANGUAGE plpgsql AS
			$$
			BEGIN
				EXECUTE format('DROP TRIGGER IF EXISTS sb_link_key_update ON %I.%I', p_schema, p_table);
				EXECUTE format('DROP TRIGGER IF EXISTS sb_link_key_delete ON %I.%I', p_schema, p_table);
				EXECUTE format(
					'CREATE TRIGGER sb_link_key_update AFTER UPDATE ON %I.%I
					 REFERENCING OLD TABLE AS link_old_rows NEW TABLE AS link_new_rows
					 FOR EACH STATEMENT EXECUTE FUNCTION public.link_key_sync()', p_schema, p_table);
				EXECUTE format(
					'CREATE TRIGGER sb_link_key_delete AFTER DELETE ON %I.%I
					 REFERENCING OLD TABLE AS link_old_rows
					 FOR EACH STATEMENT EXECUTE FUNCTION public.link_key_sync()', p_schema, p_table);
			END;
			$$;
		`,
	},
	{
		// Migration helper: converts stored row ids into key objects of p_table, keeping their order.
		FunctionName:   "link_ids_to_keys",
		FunctionParams: "p_table TEXT, p_ids BIGINT[]",
		FunctionSQL: `
			RETURNS JSONB
			LANGUAGE plpgsql AS
			$$
			DECLARE
				result JSONB;
			BEGIN
				IF p_ids IS NULL OR cardinality(p_ids) = 0 THEN
					RETURN NULL;
				END IF;
				EXECUTE format(
					'SELECT jsonb_agg(%s ORDER BY x.o) FROM unnest($1) WITH ORDINALITY AS x(id, o) JOIN %s t ON t.id = x.id',
					public.link_key_expr(p_table, 't'), p_table)
					INTO result USING p_ids;
				RETURN result;
			END;
			$$;
		`,
	},
	{
		FunctionName:   "reorder_columns_after_delete",
		FunctionParams: "p_schema_name TEXT, p_model_id TEXT, p_order_index INT",
		FunctionSQL: `
			RETURNS VOID
			LANGUAGE plpgsql AS
			$$
			DECLARE
				sql TEXT;
			BEGIN
				sql := format(
					'UPDATE %I.columns
					SET order_index = order_index - 1
					WHERE model_id = $1
					AND order_index > $2;',
					p_schema_name
				);

				EXECUTE sql USING p_model_id, p_order_index;

				RAISE NOTICE 'Reordered columns in schema %, model %, after order_index %',
					p_schema_name, p_model_id, p_order_index;
			END;
			$$;
		`,
	},
	{
		FunctionName:   "get_workspace_base_users",
		FunctionParams: "p_schema_name text, p_workspace_id uuid, p_base_id uuid",
		FunctionSQL: `
			RETURNS SETOF jsonb
			LANGUAGE plpgsql
			AS $$
			DECLARE
				sql text;
			BEGIN
				sql := format($f$
					SELECT jsonb_build_object(
						'user_id',      user_id,
						'workspace_id', workspace_id,
						'bases_ids',    bases_ids,
						'access_level', access_level      -- 👈 add this line
					)
					FROM %I.workspace_members
					WHERE workspace_id = $1::text
					AND (
							bases_ids = '*'  -- full access
							OR $2::text = ANY (
									string_to_array(
										replace(bases_ids, ' ', ''),
										','
									)
							)
					)
				$f$, p_schema_name);

				RETURN QUERY EXECUTE sql USING p_workspace_id, p_base_id;
			END;
			$$;
		`,
	},
	{
		FunctionName:   "get_users_with_role",
		FunctionParams: "",
		FunctionSQL: `
			RETURNS SETOF JSON
			LANGUAGE plpgsql STABLE AS $$
			BEGIN
				RETURN QUERY
				SELECT 
					(to_jsonb(u) || jsonb_build_object('roles', COALESCE(r.roles, '[]'::json)))::json AS user
				FROM public.users u
				LEFT JOIN (
					SELECT 
						am.user_id::uuid AS user_id,
						JSON_AGG(
							JSON_BUILD_OBJECT(
								'id', r.id,
								'name', r.name,
								'scope_level', r.scope_level,
								'priority', r.priority,
								'description', r.description,
								'role_id', am.role_id::text,
								'access_member_id', am.id
							)
						) AS roles
					FROM public.access_members am
					LEFT JOIN public.access_roles r ON r.id = am.role_id::uuid
					GROUP BY am.user_id
			) r ON r.user_id = u.id;
		END;
		$$;
	`,
	},
	{
		FunctionName:   "get_active_users_for_assign",
		FunctionParams: "",
		FunctionSQL: `
			RETURNS SETOF JSON
			LANGUAGE plpgsql STABLE AS $$
			BEGIN
				RETURN QUERY
				SELECT 
					(to_jsonb(u) || jsonb_build_object('roles', COALESCE(r.roles, '[]'::json)))::json AS user
				FROM public.users u
				LEFT JOIN (
					SELECT 
						am.user_id::uuid AS user_id,
						JSON_AGG(
							JSON_BUILD_OBJECT(
								'id', ar.id,
								'name', ar.name,
								'scope_level', ar.scope_level,
								'priority', ar.priority,
								'description', ar.description,
								'role_id', am.role_id::text,
								'access_member_id', am.id
							)
						) AS roles
					FROM public.access_members am
					LEFT JOIN public.access_roles ar ON am.role_id::uuid = ar.id
					GROUP BY am.user_id
			) r ON r.user_id = u.id
			WHERE u.status = 'active'
			AND u.id NOT IN (
				SELECT DISTINCT am.user_id::uuid
				FROM public.access_members am
				LEFT JOIN public.access_roles ar ON am.role_id::uuid = ar.id
				WHERE ar.name IN ('owner', 'co-owner')
			)
			ORDER BY u.display_name ASC;
		END;
		$$;
	`,
	},
	{
		FunctionName:   "get_workspace_members_with_role",
		FunctionParams: "p_workspace_id text",
		FunctionSQL: `
		RETURNS SETOF JSON
		LANGUAGE plpgsql STABLE AS $$
		DECLARE
			v_workspace_id uuid;
		BEGIN
			v_workspace_id := p_workspace_id::uuid;
			RETURN QUERY
			SELECT 
				(to_jsonb(u) || jsonb_build_object('roles', COALESCE(r.roles, '[]'::json)))::json AS user
			FROM public.users u
			INNER JOIN (
				-- Get all users who have any access to this workspace from access_members
				-- Either scope_id matches (workspace-level) OR workspace_id matches (base-level)
				SELECT DISTINCT am.user_id::uuid
				FROM public.access_members am
				LEFT JOIN public.access_roles ar ON am.role_id::uuid = ar.id
				WHERE ar.name NOT IN ('owner', 'co-owner')
				  AND (
					  -- Workspace-level access: scope_type='workspace' and scope_id is this workspace
					  (am.scope_type = 'workspace' AND am.scope_id::uuid = v_workspace_id)
					  OR
					  -- Base-level access: scope_type='base' and the base belongs to this workspace
					  (am.scope_type = 'base' AND EXISTS (
						  SELECT 1 FROM public.bases b 
						  WHERE b.id::uuid = am.scope_id::uuid 
						  AND b.workspace_id::uuid = v_workspace_id
					  ))
				  )
				UNION
				-- Get users from workspace_members table
				SELECT DISTINCT wm.user_id::uuid
				FROM public.workspace_members wm
				WHERE wm.workspace_id::uuid = v_workspace_id
			) members ON members.user_id = u.id
			LEFT JOIN (
				-- Aggregate all roles for each user
				SELECT 
					am.user_id::uuid AS user_id,
					JSON_AGG(
						JSON_BUILD_OBJECT(
							'id', r.id,
							'name', r.name,
							'scope_level', r.scope_level,
							'priority', r.priority,
							'description', r.description,
							'role_id', am.role_id::text,
							'access_member_id', am.id,
							'scope_id', am.scope_id,
							'scope_type', am.scope_type
						)
					) AS roles
				FROM public.access_members am
				LEFT JOIN public.access_roles r ON r.id = am.role_id::uuid
				WHERE r.name NOT IN ('owner', 'co-owner')
				  AND (
					  (am.scope_type = 'workspace' AND am.scope_id::uuid = v_workspace_id)
					  OR
					  (am.scope_type = 'base' AND EXISTS (
						  SELECT 1 FROM public.bases b 
						  WHERE b.id::uuid = am.scope_id::uuid 
						  AND b.workspace_id::uuid = v_workspace_id
					  ))
				  )
				GROUP BY am.user_id
				
				UNION ALL
				
				-- Get roles from workspace_members table
				SELECT 
					wm.user_id::uuid AS user_id,
					JSON_AGG(
						JSON_BUILD_OBJECT(
							'id', r.id,
							'name', r.name,
							'scope_level', r.scope_level,
							'priority', r.priority,
							'description', r.description,
							'role_id', r.id::text,
							'access_member_id', wm.id::text,
							'scope_id', wm.workspace_id::text,
							'scope_type', 'workspace'
						)
					) AS roles
				FROM public.workspace_members wm
				LEFT JOIN public.access_roles r ON r.name = wm.access_level
				WHERE wm.workspace_id::uuid = v_workspace_id
				GROUP BY wm.user_id
			) r ON r.user_id = u.id
			WHERE u.status = 'active';
		END;
		$$;
	`,
	},
	{
		FunctionName:   "get_base_members_with_role",
		FunctionParams: "p_base_id text",
		FunctionSQL: `
		RETURNS SETOF JSON
		LANGUAGE plpgsql STABLE AS $$
		DECLARE
			v_base_id uuid;
		BEGIN
			v_base_id := p_base_id::uuid;
			RETURN QUERY
			SELECT 
				(to_jsonb(u) || jsonb_build_object('roles', COALESCE(r.roles, '[]'::json)))::json AS user
			FROM public.users u
			INNER JOIN (
				-- Get users from access_members with base scope
				SELECT DISTINCT am.user_id::uuid
				FROM public.access_members am
				WHERE am.scope_type = 'base' AND am.scope_id::uuid = v_base_id
			) members ON members.user_id = u.id
			LEFT JOIN (
				SELECT 
					am.user_id::uuid AS user_id,
					JSON_AGG(
						JSON_BUILD_OBJECT(
							'id', r.id,
							'name', r.name,
							'scope_level', r.scope_level,
							'priority', r.priority,
							'description', r.description,
							'role_id', am.role_id::text,
							'access_member_id', am.id,
							'scope_id', am.scope_id,
							'scope_type', am.scope_type
						)
					) AS roles
				FROM public.access_members am
				LEFT JOIN public.access_roles r ON r.id = am.role_id::uuid
				WHERE am.scope_id::uuid = v_base_id AND am.scope_type = 'base'
				GROUP BY am.user_id
			) r ON r.user_id = u.id
			WHERE u.status = 'active';
		END;
		$$;
	`,
	},
	{
		FunctionName:   "get_user_role_by_id",
		FunctionParams: "p_user_id text",
		FunctionSQL: `
	RETURNS SETOF JSON
	LANGUAGE plpgsql STABLE AS $$
	DECLARE
		v_user_id uuid;
	BEGIN
		v_user_id := p_user_id::uuid;
		RETURN QUERY
		SELECT 
			JSON_BUILD_OBJECT(
				'user_id', u.id,
				'role_name', ar.name
			)::json AS result
		FROM public.users u
		INNER JOIN public.access_members am ON am.user_id::uuid = u.id
		INNER JOIN public.access_roles ar ON ar.id = am.role_id::uuid
		WHERE u.id = v_user_id;
	END;
	$$;
`,
	},
	{
		FunctionName:   "get_user_permission_access",
		FunctionParams: "p_schema_name text, p_user_id text, p_scope_type text, p_scope_id text, p_resource_code text, p_action_code text",
		FunctionSQL: `
			RETURNS JSON
			LANGUAGE plpgsql STABLE AS $$
			DECLARE
				sql TEXT;
				has_access BOOLEAN := FALSE;
			BEGIN
				sql := format(
					'WITH candidate_access AS (
						SELECT am.role_id::uuid AS role_id
						FROM %I.access_members am
						WHERE am.user_id = $1
						  AND am.scope_type = $2
						  AND ($3 IS NULL OR $3 = '''' OR am.scope_id = $3)
					)
					SELECT EXISTS (
						SELECT 1
						FROM candidate_access ca
						JOIN %I.role_permissions rp ON rp.role_id = ca.role_id
						JOIN %I.permissions p ON rp.permission_id = p.id
						JOIN %I.actions a ON p.action_id = a.id
						JOIN %I.resources r ON p.resource_id = r.id
						WHERE r.code = $4 AND a.code = $5
					)',
					p_schema_name,
					p_schema_name,
					p_schema_name,
					p_schema_name,
					p_schema_name
				);

				EXECUTE sql INTO has_access USING p_user_id, p_scope_type, p_scope_id, p_resource_code, p_action_code;

				RETURN json_build_object('has_access', COALESCE(has_access, FALSE));
			END;
			$$;
		`,
	},
	{
		FunctionName:   "bulk_update",
		FunctionParams: "p_schema_name TEXT, p_table_name TEXT, p_column_name TEXT, p_data JSONB",
		FunctionSQL: `
			RETURNS VOID AS $$
			DECLARE
				rec JSONB;
				update_query TEXT;
				full_table_name TEXT;
				row_id TEXT;
				col_value TEXT;
			BEGIN
				full_table_name := format('%I.%I', p_schema_name, p_table_name);
				
				FOR rec IN SELECT * FROM jsonb_array_elements(p_data)
				LOOP
					row_id := rec->>'id';
					col_value := rec->>'value';

					update_query := format(
						'UPDATE %s SET %I = %L WHERE id = %L',
						full_table_name,
						p_column_name,
						col_value,
						row_id
					);

					EXECUTE update_query;
				END LOOP;
			END;
			$$ LANGUAGE plpgsql;
		`,
	},
	{
		FunctionName:   "reset_column",
		FunctionParams: "p_schema_name TEXT, p_table_name TEXT, p_column_name TEXT",
		FunctionSQL: `
			RETURNS VOID AS $$
			DECLARE
				query TEXT;
				full_table_name TEXT;
			BEGIN
				full_table_name := format('%I.%I', p_schema_name, p_table_name);
				
				query := format(
					'UPDATE %s SET %I = NULL',
					full_table_name,
					p_column_name
				);

				EXECUTE query;
			END;
			$$ LANGUAGE plpgsql;
		`,
	},
	{
		FunctionName:   "bulk_update_by_columns",
		FunctionParams: "p_schema_name TEXT, p_table_name TEXT, p_data JSONB",
		FunctionSQL: `
			RETURNS VOID AS $$
			DECLARE
				full_table_name TEXT;
				col_name TEXT;
				update_sql TEXT;
			BEGIN
				full_table_name := format('%I.%I', p_schema_name, p_table_name);

				FOR col_name IN
					SELECT DISTINCT elem->>'column'
					FROM jsonb_array_elements(p_data) AS elem
				LOOP
					update_sql := format(
						'UPDATE %s AS t
						 SET %I = u.value
						 FROM (
						 	SELECT (elem->>''id'')::INT AS id, elem->>''value'' AS value
						 	FROM jsonb_array_elements($1) AS elem
						 	WHERE elem->>''column'' = $2
						 ) AS u
						 WHERE t.id = u.id
						   AND t.%I IS DISTINCT FROM u.value',
						full_table_name,
						col_name,
						col_name
					);

					EXECUTE update_sql USING p_data, col_name;
				END LOOP;
			END;
			$$ LANGUAGE plpgsql;
		`,
	},
}

// DroppedFunctions are signatures replaced by a new signature or no longer used; they are dropped
// before DefinedFunctions are installed so stale overloads are not left behind.
var DroppedFunctions = []string{
	"link_apply_change(TEXT, TEXT, BOOLEAN, BIGINT, BIGINT, BOOLEAN, TEXT)",
	"link_rows_holding(TEXT, TEXT, BOOLEAN, BIGINT)",
	// Link cleanup on row delete is done by the link_key_sync trigger
	"remove_link_back_references(JSON[], BIGINT[], TEXT)",
	// Placeholder key values for new rows are no longer generated; CASCADE drops its sb_key_defaults triggers
	"key_fill_defaults() CASCADE",
}

// LinkColumnsToMigrateQuery lists link columns still stored as row ids (INT / INT[]) with their table and target table
const LinkColumnsToMigrateQuery = `
	SELECT c.id::text, c.column_name, m.alias, COALESCE(tm.alias, ''), t.typcategory = 'A'
	FROM %[1]s.columns c
	JOIN %[1]s.models m ON m.id::text = c.model_id::text
	LEFT JOIN %[1]s.models tm ON tm.id::text = c.meta->'relation'->>'with'
	JOIN pg_attribute a ON a.attrelid = to_regclass(format('%%I.%%I', '%[2]s', m.alias)) AND a.attname = c.column_name
	JOIN pg_type t ON t.oid = a.atttypid
	WHERE c.uidt = 'links' AND NOT COALESCE(c.deleted, FALSE)
	  AND a.atttypid IN ('integer'::regtype, 'bigint'::regtype, 'integer[]'::regtype, 'bigint[]'::regtype)`

// InstallKeySyncTriggersQuery installs the link key-sync triggers on every table that does not have them yet
const InstallKeySyncTriggersQuery = `
	SELECT public.link_install_key_sync('%[2]s', m.alias)
	FROM %[1]s.models m
	WHERE to_regclass(format('%%I.%%I', '%[2]s', m.alias)) IS NOT NULL
	  AND NOT EXISTS (
		SELECT 1 FROM pg_trigger
		WHERE tgrelid = to_regclass(format('%%I.%%I', '%[2]s', m.alias)) AND tgname = 'sb_link_key_delete')`
