## API Documentation - Database Schema Extractor Service

This document describes the HTTP API exposed by the Database Schema Extractor service.

The service converts natural-language descriptions of database tables into a normalized JSON schema.

---

## Base URL

- **Local development**: `http://localhost:8080`

All endpoints documented below are relative to this base URL.

---

## Endpoint: Extract Schema

- **URL**: `/extract-schema`  
- **Method**: `POST`  
- **Content-Type**: `application/json`  
- **Description**:  
  Takes a natural-language prompt describing one or more database tables and returns a JSON description of the inferred schema.

### Request

- **Headers**
  - **Content-Type**: `application/json`

- **Body**

  ```json
  {
    "prompt": "string"
  }
  ```

  - **Fields**
    - **prompt** (string, required):  
      Free-form natural language description of the desired database tables and their fields/columns.

  - **Example**

    ```json
    {
      "prompt": "Create a users table with id, username, email, password, and created_at"
    }
    ```

### Successful Response (200 OK)

If the OpenAI response is valid JSON matching the internal schema model, the API returns it directly.

- **Body schema**

  ```json
  {
    "tables": [
      {
        "name": "string",
        "fields": [
          {
            "name": "string",
            "type": "string"
          }
        ]
      }
    ]
  }
  ```

  - **tables** (array of `Table`, required): List of inferred tables.
    - **name** (string, required): Table name (lowercase snake_case).
    - **fields** (array of `Field`, required): List of fields/columns for the table.
      - **name** (string, required): Field name (lowercase snake_case, exactly as provided by user).
      - **type** (string, required): Semantic data type. Supported types include: `"text"`, `"longText"`, `"number"`, `"decimal"`, `"boolean"`, `"currency"`, `"percent"`, `"duration"`, `"year"`, `"date"`, `"datetime"`, `"time"`, `"email"`, `"phoneNumber"`, `"url"`, `"select"`, `"multiSelect"`, `"rating"`, `"user"`, `"button"`, `"json"`, `"uuid"`, `"createdTime"`, `"lastModifiedTime"`. Field names with `"id"` default to type `"uuid"`.

- **Example**

  ```json
  {
    "tables": [
      {
        "name": "users",
        "fields": [
          {
            "name": "id",
            "type": "uuid"
          },
          {
            "name": "username",
            "type": "text"
          },
          {
            "name": "email",
            "type": "email"
          },
          {
            "name": "password",
            "type": "text"
          },
          {
            "name": "created_at",
            "type": "createdTime"
          }
        ]
      }
    ]
  }
  ```

### Fallback Successful Response (200 OK)

If the OpenAI response is **not** valid JSON but still succeeds, the service wraps the raw string response in an object:

- **Body schema**

  ```json
  {
    "result": "string"
  }
  ```

  - **result** (string, required): Raw textual response returned by the model.

### Error Responses

- **400 Bad Request**

  - Conditions:
    - Request body is not valid JSON.
    - `prompt` field is missing or empty.

  - Examples:

    ```json
    {
      "error": "invalid JSON body"
    }
    ```

    or

    ```json
    {
      "error": "prompt is required"
    }
    ```

  (Exact error body may be plain text depending on HTTP client; the status code is the primary contract.)

- **405 Method Not Allowed**

  - Condition: Any HTTP method other than `POST` is used.
  - Response body: `"method not allowed"` (plain text).

- **500 Internal Server Error**

  - Condition: Call to OpenAI API fails or cannot produce a valid schema.
  - Response body: `"failed to extract schema"` (plain text).

---

## Data Models (Go Types)

These structs represent the JSON schema when the response is valid JSON:

- **Field**

  ```json
  {
    "name": "string",
    "type": "string"
  }
  ```

- **Table**

  ```json
  {
    "name": "string",
    "fields": [Field]
  }
  ```

- **SchemaResponse**

  ```json
  {
    "tables": [Table]
  }
  ```

- **ExtractSchemaRequest**

  ```json
  {
    "prompt": "string"
  }
  ```

- **ExtractSchemaAPIResponse** (fallback wrapper)

  ```json
  {
    "result": "string"
  }
  ```

---

## Authentication & Configuration

- **Authentication**:  
  This API does **not** require client-side authentication, but the server must be started with a valid `OPENAI_API_KEY` configured via:
  - `.env` file with `OPENAI_API_KEY=your-api-key`, or
  - Environment variable `OPENAI_API_KEY` on the host.

- **Model & Prompting**:
  - Uses OpenAI Chat Completions with model `gpt-4.1` (via the `openai-go` client and `ChatModelGPT4o` constant).
  - System behavior is controlled by the `database_meta.prompt` file, which defines how schemas are inferred and formatted.
  - The prompt uses semantic data types (e.g., `text`, `email`, `uuid`, `datetime`) rather than SQL-specific types.
  - Field names are preserved exactly as provided by the user, and table/field names use lowercase snake_case format.
  - Fields named `id` automatically use type `uuid`.


