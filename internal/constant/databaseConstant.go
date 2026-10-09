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
	{
		// Replaces get_table_data_with_relation. Each relation_data entry carries its own output aliases
		// (the lookup column names) and whether the link column is an array, so two lookups of the same
		// field through different links (including both directions of a self-link) never collide.
		// relation_data entry: {source_column_name, target_table_name, target_column_name, relation,
		//                       is_array, targets: [{column, alias}]}
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
				target_table_name TEXT;
				target_column_name TEXT;
				relation TEXT;
				is_array BOOLEAN;
				expr TEXT;
				relation_sql TEXT := '';
				query TEXT;
				result JSON;
			BEGIN
				FOR rel IN SELECT * FROM unnest(COALESCE(relation_data, ARRAY[]::JSON[]))
				LOOP
					source_column_name := rel->>'source_column_name';
					target_table_name  := rel->>'target_table_name';
					target_column_name := rel->>'target_column_name';
					relation           := rel->>'relation';
					is_array           := COALESCE((rel->>'is_array')::BOOLEAN, FALSE);

					FOR target IN SELECT * FROM json_array_elements(rel->'targets')
					LOOP
						IF is_array THEN
							expr := format(
								'(SELECT COALESCE(JSON_AGG(t.%I), ''[]''::JSON) FROM %I.%I t WHERE t.%I = ANY(s.%I)) AS %I',
								target->>'column', schema_name, target_table_name, target_column_name, source_column_name,
								target->>'alias'
							);
						ELSIF relation IN ('has-many', 'many-to-many') THEN
							expr := format(
								'(SELECT COALESCE(JSON_AGG(t.%I), ''[]''::JSON) FROM %I.%I t WHERE t.%I = s.%I) AS %I',
								target->>'column', schema_name, target_table_name, target_column_name, source_column_name,
								target->>'alias'
							);
						ELSE
							expr := format(
								'(SELECT t.%I FROM %I.%I t WHERE t.%I = s.%I LIMIT 1) AS %I',
								target->>'column', schema_name, target_table_name, target_column_name, source_column_name,
								target->>'alias'
							);
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
	// --- Link functions --------------------------------------------------------------------------
	// Link/unlink and row-delete cleanup run as single function calls, so each operation is atomic
	// without a client-side transaction. p_table values are already-quoted "schema"."table" names
	// taken from metadata; column names are quoted here with %I.
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
		// Adds (p_link) or removes p_value in one row's link column. INT[] columns are updated in place.
		FunctionName:   "link_apply_change",
		FunctionParams: "p_table TEXT, p_column TEXT, p_is_array BOOLEAN, p_row BIGINT, p_value BIGINT, p_link BOOLEAN, p_updated_by TEXT",
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
						'%1$I = CASE WHEN $1::int = ANY(COALESCE(%1$I, ''{}''::int[])) THEN %1$I ELSE array_append(COALESCE(%1$I, ''{}''::int[]), $1::int) END',
						p_column);
				ELSIF p_is_array THEN
					set_expr := format('%1$I = array_remove(%1$I, $1::int)', p_column);
				ELSIF p_link THEN
					set_expr := format('%I = $1::int', p_column);
				ELSE
					set_expr := format('%I = NULL', p_column);
					where_expr := format('id = $2 AND %I = $1::int', p_column);
				END IF;

				IF COALESCE(p_updated_by, '') <> '' THEN
					audit := audit || ', last_modified_by = $4';
				END IF;

				EXECUTE format('UPDATE %s SET %s%s WHERE %s', p_table, set_expr, audit, where_expr)
					USING p_value, p_row, (now() AT TIME ZONE 'UTC'), p_updated_by;
			END;
			$$;
		`,
	},
	{
		// Returns (and locks) the rows whose link column contains p_value.
		FunctionName:   "link_rows_holding",
		FunctionParams: "p_table TEXT, p_column TEXT, p_is_array BOOLEAN, p_value BIGINT",
		FunctionSQL: `
			RETURNS SETOF BIGINT
			LANGUAGE plpgsql AS
			$$
			BEGIN
				IF p_is_array THEN
					RETURN QUERY EXECUTE format('SELECT id::bigint FROM %s WHERE $1::int = ANY(%I) FOR UPDATE', p_table, p_column)
						USING p_value;
				ELSE
					RETURN QUERY EXECUTE format('SELECT id::bigint FROM %s WHERE %I = $1::int FOR UPDATE', p_table, p_column)
						USING p_value;
				END IF;
			END;
			$$;
		`,
	},
	{
		// Would making p_parent the parent of p_child create a loop? p_column is the INT column that
		// points from a child to its parent.
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
					'WITH RECURSIVE ancestors(id) AS (
						SELECT %1$I::bigint FROM %2$s WHERE id = $1
						UNION
						SELECT t.%1$I::bigint FROM %2$s t JOIN ancestors a ON t.id = a.id
					)
					SELECT EXISTS (SELECT 1 FROM ancestors WHERE id = $2)', p_column, p_table)
					INTO result USING p_parent, p_child;
				RETURN result;
			END;
			$$;
		`,
	},
	{
		// Would adding p_child to p_parent's children (an INT[] column) create a loop?
		FunctionName:   "link_cycle_children",
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
					'WITH RECURSIVE ancestors(id) AS (
						SELECT id::bigint FROM %2$s WHERE $1::int = ANY(%1$I)
						UNION
						SELECT t.id::bigint FROM %2$s t JOIN ancestors a ON a.id::int = ANY(t.%1$I)
					)
					SELECT EXISTS (SELECT 1 FROM ancestors WHERE id = $2)', p_column, p_table)
					INTO result USING p_parent, p_child;
				RETURN result;
			END;
			$$;
		`,
	},
	{
		// Links or unlinks two rows. Returns 'ok', 'source_not_found', 'target_not_found' or 'cycle';
		// every non-ok status is returned before anything is written.
		FunctionName: "link_rows",
		FunctionParams: "p_relation_type TEXT, p_single_column BOOLEAN, p_is_self BOOLEAN, p_action TEXT, " +
			"p_source_table TEXT, p_source_column TEXT, p_source_is_array BOOLEAN, " +
			"p_target_table TEXT, p_target_column TEXT, p_target_is_array BOOLEAN, " +
			"p_source_row BIGINT, p_target_row BIGINT, p_updated_by TEXT",
		FunctionSQL: `
			RETURNS TEXT
			LANGUAGE plpgsql AS
			$$
			DECLARE
				is_link BOOLEAN := p_action = 'link';
				holder BIGINT;
				has_cycle BOOLEAN;
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

				-- One-way self-link: only the edited row's cell changes.
				IF p_single_column THEN
					IF NOT is_link THEN
						PERFORM public.link_apply_change(p_source_table, p_source_column, p_source_is_array, p_source_row, p_target_row, FALSE, p_updated_by);
						RETURN 'ok';
					END IF;
					IF p_relation_type = 'has-many'
						AND public.link_cycle_children(p_source_table, p_source_column, p_source_row, p_target_row) THEN
						RETURN 'cycle';
					END IF;
					IF p_relation_type IN ('one-to-one', 'has-many') THEN
						FOR holder IN SELECT * FROM public.link_rows_holding(p_source_table, p_source_column, p_source_is_array, p_target_row) LOOP
							IF holder <> p_source_row THEN
								PERFORM public.link_apply_change(p_source_table, p_source_column, p_source_is_array, holder, p_target_row, FALSE, p_updated_by);
							END IF;
						END LOOP;
					END IF;
					PERFORM public.link_apply_change(p_source_table, p_source_column, p_source_is_array, p_source_row, p_target_row, TRUE, p_updated_by);
					RETURN 'ok';
				END IF;

				IF NOT is_link THEN
					PERFORM public.link_apply_change(p_source_table, p_source_column, p_source_is_array, p_source_row, p_target_row, FALSE, p_updated_by);
					PERFORM public.link_apply_change(p_target_table, p_target_column, p_target_is_array, p_target_row, p_source_row, FALSE, p_updated_by);
					RETURN 'ok';
				END IF;

				-- Two-column self-link has-many: the INT side points from a child to its parent.
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
					FOR holder IN SELECT * FROM public.link_rows_holding(p_source_table, p_source_column, p_source_is_array, p_target_row) LOOP
						PERFORM public.link_apply_change(p_source_table, p_source_column, p_source_is_array, holder, p_target_row, FALSE, p_updated_by);
						PERFORM public.link_apply_change(p_target_table, p_target_column, p_target_is_array, p_target_row, holder, FALSE, p_updated_by);
					END LOOP;
				END IF;
				IF p_relation_type = 'one-to-one' OR (p_relation_type = 'has-many' AND p_target_is_array) THEN
					FOR holder IN SELECT * FROM public.link_rows_holding(p_target_table, p_target_column, p_target_is_array, p_source_row) LOOP
						PERFORM public.link_apply_change(p_target_table, p_target_column, p_target_is_array, holder, p_source_row, FALSE, p_updated_by);
						PERFORM public.link_apply_change(p_source_table, p_source_column, p_source_is_array, p_source_row, holder, FALSE, p_updated_by);
					END LOOP;
				END IF;

				PERFORM public.link_apply_change(p_source_table, p_source_column, p_source_is_array, p_source_row, p_target_row, TRUE, p_updated_by);
				PERFORM public.link_apply_change(p_target_table, p_target_column, p_target_is_array, p_target_row, p_source_row, TRUE, p_updated_by);
				RETURN 'ok';
			END;
			$$;
		`,
	},
	{
		// Removes every reference to p_row_ids from the given partner link columns.
		// p_partners entry: {"table": "\"schema\".\"table\"", "column": "...", "is_array": true}
		FunctionName:   "remove_link_back_references",
		FunctionParams: "p_partners JSON[], p_row_ids BIGINT[], p_updated_by TEXT",
		FunctionSQL: `
			RETURNS VOID
			LANGUAGE plpgsql AS
			$$
			DECLARE
				partner JSON;
				partner_is_array BOOLEAN;
				row_id BIGINT;
				holder BIGINT;
			BEGIN
				FOR partner IN SELECT * FROM unnest(COALESCE(p_partners, ARRAY[]::JSON[])) LOOP
					partner_is_array := COALESCE((partner->>'is_array')::BOOLEAN, FALSE);
					FOR row_id IN SELECT * FROM unnest(COALESCE(p_row_ids, ARRAY[]::BIGINT[])) LOOP
						FOR holder IN SELECT * FROM public.link_rows_holding(partner->>'table', partner->>'column', partner_is_array, row_id) LOOP
							PERFORM public.link_apply_change(partner->>'table', partner->>'column', partner_is_array, holder, row_id, FALSE, p_updated_by);
						END LOOP;
					END LOOP;
				END LOOP;
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
