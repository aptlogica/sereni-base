# AI Service API Documentation

This document covers the standalone AI schema extractor service and the newly added AI-assisted routes in the main SereniBase API.

## 1. Standalone AI Schema Extractor Service

The service in `services/Ai Services` runs as a separate HTTP server and returns AI-generated schema output that is consumed by the main application.

### Base URL

- Local default: `http://localhost:8888`
- API prefix: `/api/v1`

### Environment

- `OPENAI_API_KEY` is required.
- `SERVER_PORT` or `PORT` can override the default port.

### 1.1 Extract Schema

- **Method:** `POST`
- **URL:** `/api/v1/extract-schema`
- **Content-Type:** `application/json`

#### Request

```json
{
  "prompt": "Create a customer table with name, email, phone and created time"
}
```

#### Response

If the model returns valid JSON, the response is returned as structured JSON.

```json
{
  "tables": [
    {
      "name": "customer",
      "fields": [
        {
          "name": "name",
          "type": "text"
        }
      ]
    }
  ]
}
```

If the model returns plain text, the service wraps it as:

```json
{
  "result": "string"
}
```

### 1.2 Extract CSV Schema

- **Method:** `POST`
- **URL:** `/api/v1/extract-schema-csv`
- **Content-Type:** `application/json`

#### Request

```json
{
  "prompt": "Generate CSV data for 10 rows",
  "row_count": 10,
  "table": {
    "name": "customer"
  }
}
```

#### Response

- `text/csv`
- Returns generated CSV content.

### 1.3 Extract Multiple Base Schema

- **Method:** `POST`
- **URL:** `/api/v1/extract-schema-multiple`
- **Content-Type:** `application/json`

#### Request

```json
{
  "prompt": "Create a CRM base with customers, contacts, and opportunities"
}
```

#### Response

Same response shape as `/extract-schema`, but intended for multi-table base generation.

### 1.4 Chat Proxy

- **Method:** `POST`
- **URL:** `/api/chat`
- **Alias:** `/api/v1/chat`
- **Content-Type:** `application/json`

#### Request

```json
{
  "messages": [
    { "role": "user", "content": "Summarize this project in one sentence." }
  ],
  "model": "gpt-4o-mini",
  "temperature": 0.7
}
```

#### Response

```json
{
  "reply": "..."
}
```

---

## 2. Newly Added Main API Routes

These routes are exposed by the main SereniBase application under `/api/v1`.

### Authentication

All routes below require:

```http
Authorization: Bearer <access_token>
```

### 2.1 Preview AI Base

- **Method:** `POST`
- **URL:** `/api/v1/base/create/ai`
- **Content-Type:** `application/json`

This route previews an AI-generated base schema without creating anything in the database.

#### Request

```json
{
  "prompt": "Create a CRM base with customer, contact and opportunity tables"
}
```

#### Response

```json
{
  "success": true,
  "message": "Base fetched successfully",
  "data": {
    "base_name": "crm_base",
    "relations": [],
    "tables": [
      {
        "name": "customer",
        "fields": [
          {
            "name": "first_name",
            "type": "text",
            "meta": {}
          }
        ]
      }
    ]
  }
}
```

### 2.2 Apply AI Base

- **Method:** `POST`
- **URL:** `/api/v1/base/create/ai/apply`
- **Content-Type:** `application/json`

This route creates the base and then creates the tables described by the AI schema.

#### Request

```json
{
  "workspace_id": "workspace-uuid",
  "sample_data": false,
  "row": 0,
  "base_name": "crm_base",
  "relations": [],
  "tables": [
    {
      "name": "customer",
      "fields": [
        {
          "name": "first_name",
          "type": "text",
          "meta": {}
        },
        {
          "name": "email",
          "type": "email",
          "meta": {}
        }
      ]
    }
  ]
}
```

#### Required fields

- `workspace_id`
- `base_name`
- `tables`

#### Optional fields

- `sample_data`
- `row`
- `relations`

#### Response

```json
{
  "success": true,
  "message": "table created",
  "data": "okay"
}
```

### 2.3 Preview AI Table

- **Method:** `POST`
- **URL:** `/api/v1/table/ai`
- **Content-Type:** `application/json`

This route previews AI-generated table schema without creating tables.

#### Request

```json
{
  "prompt": "Create a products table with name, description, price and stock"
}
```

#### Response

```json
{
  "success": true,
  "message": "Table fetched successfully",
  "data": {
    "tables": [
      {
        "name": "product",
        "fields": [
          {
            "name": "product_name",
            "type": "text",
            "meta": {}
          }
        ]
      }
    ],
    "relations": []
  }
}
```

### 2.4 Apply AI Table

- **Method:** `POST`
- **URL:** `/api/v1/table/ai/apply`
- **Content-Type:** `application/json`

This route creates one or more tables from an edited AI schema.

#### Request

```json
{
  "base_id": "base-uuid",
  "workspace_id": "workspace-uuid",
  "sample_data": false,
  "row": 0,
  "tables": [
    {
      "name": "product",
      "fields": [
        {
          "name": "product_name",
          "type": "text",
          "meta": {}
        },
        {
          "name": "price",
          "type": "currency",
          "meta": {}
        }
      ]
    }
  ]
}
```

#### Required fields

- `base_id`
- `workspace_id`
- `tables`

#### Optional fields

- `sample_data`
- `row`

#### Response

```json
{
  "success": true,
  "message": "table created",
  "data": [
    {
      "import_stats": {
        "total_rows": 0,
        "total_columns": 0,
        "error_rows": 0
      }
    }
  ]
}
```

---

## 3. Data Shapes

### AI Base Schema

```json
{
  "base_name": "string",
  "relations": [
    {
      "type": "string",
      "source_table": "string",
      "target_table": "string"
    }
  ],
  "tables": [
    {
      "name": "string",
      "fields": [
        {
          "name": "string",
          "type": "string",
          "meta": {}
        }
      ]
    }
  ]
}
```

### AI Table Schema

```json
{
  "tables": [
    {
      "name": "string",
      "fields": [
        {
          "name": "string",
          "type": "string",
          "meta": {}
        }
      ]
    }
  ],
  "relations": []
}
```

---

## 4. Notes

- `Preview` routes only generate schema output.
- `Apply` routes create records in the database.
- The main API routes are protected by the normal SereniBase authentication middleware.
- The AI service is a helper service used by the application to generate schemas from natural language prompts.
