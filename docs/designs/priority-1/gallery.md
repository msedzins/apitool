# Priority 1 screen gallery

Draft for review. These are static terminal mockups, not implemented screens.

[Interaction specification](README.md)

## Screen index

- [Request editor: query parameters](#01-params)

- [Request editor: headers](#02-headers)

- [Body: structured JSON](#03-json)

- [Body: invalid JSON text](#04-json-invalid)

- [Body: raw payload](#05-raw)

- [Request identity and effective settings](#06-settings)

- [Auth: inherited OAuth](#07-auth-inherited)

- [Auth: request override](#08-auth-override)

- [Auth: explicit token reveal](#09-auth-reveal)

- [Execution: sending current draft](#10-sending)

- [Response: completed JSON exchange](#11-response)

- [Response: completed HTTP error](#12-http-error)

- [Response: cached result](#13-cached)

- [Response: no content](#14-empty-body)

- [Diagnostics: missing process variable](#15-validation)

- [Diagnostics: transport timeout](#16-transport)

- [Diagnostics: redacted request log](#17-log)

- [Diagnostics: invalid selected definition](#18-invalid-definition)

- [Unsaved changes: navigation](#19-dirty-navigation)

- [Unsaved changes: save failed](#20-save-failure)

- [Dangerous send: explicit confirmation](#21-confirm-send)

<a id="01-params"></a>

## Request editor: query parameters

Typed table editing; display parameter values as text. Commit a cell with Enter; Esc reverts that cell.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │[Params] | Headers | Auth | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Name: Create payment *                                                  │
│  > POST create          │Query parameters (Enter edit; Insert add; Delete remove)                │
│    ! broken             │  Key           Value                                                   │
│                         │> dry_run       false                                                   │
│▸ users                  │  api_version   {{api_version}}                                         │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │No response yet.                                                        │
│                         │Send executes the draft; Save writes the definition.                    │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │[Params] | Headers | Auth | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Name: Create payment *                                   │
│  > POST create     │Query parameters (Enter edit; Insert add; Delete remove) │
│    ! broken        │  Key           Value                                    │
│                    │> dry_run       false                                    │
│▸ users             │  api_version   {{api_version}}                          │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │No response yet.                                         │
│                    │Send executes the draft; Save writes the definition.     │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="02-headers"></a>

## Request editor: headers

Case-insensitive duplicate header names are validation errors. Mask sensitive resolved values; reference expressions remain editable.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | [Headers] | Auth | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Name: Create payment *                                                  │
│  > POST create          │Headers (Enter edit; Insert add; Delete remove)                         │
│    ! broken             │  Key                Value                                              │
│                         │> Accept             application/json                                   │
│▸ users                  │  Content-Type       application/json                                   │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │No response yet.                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | [Headers] | Auth | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Name: Create payment *                                   │
│  > POST create     │Headers (Enter edit; Insert add; Delete remove)          │
│    ! broken        │  Key                Value                               │
│                    │> Accept             application/json                    │
│▸ users             │  Content-Type       application/json                    │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │No response yet.                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="03-json"></a>

## Body: structured JSON

Structured JSON supports nested objects and arrays, typed values and field rename. Expand with arrows; edit a leaf with Enter. Text view edits the same JSON value.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | Text                               │
│  > POST create          │> amount     number   1200                                              │
│    ! broken             │  currency   string   PLN                                               │
│                         │  metadata   object   {...}                                             │
│▸ users                  │Enter edit value; Insert add field; Delete remove                       │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │No response yet.                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | Text                │
│  > POST create     │> amount     number   1200                               │
│    ! broken        │  currency   string   PLN                                │
│                    │  metadata   object   {...}                              │
│▸ users             │Enter edit value; Insert add field; Delete remove        │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │No response yet.                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="04-json-invalid"></a>

## Body: invalid JSON text

Preserve invalid text in the draft. Block Structured view until parseable. Pretty-print parses first and preserves text on error.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send disabled]  [Save] * Unsaved                                       │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | [Text]                             │
│  > POST create          │1 {                                                                     │
│    ! broken             │2   "amount": 1200,                                                     │
│                         │3   "currency": "PLN",                                                  │
│▸ users                  │4 }                                                                     │
│                         │! Line 4, column 1: trailing comma                                      │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │Validation blocked: JSON body is invalid.                               │
│                         │Fix the text before Save or Send.                                       │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send disabled]  [Save] * Unsaved                        │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | [Text]              │
│  > POST create     │1 {                                                      │
│    ! broken        │2   "amount": 1200,                                      │
│                    │3   "currency": "PLN",                                   │
│▸ users             │4 }                                                      │
│                    │! Line 4, column 1: trailing comma                       │
│                    ├─────────────────────────────────────────────────────────┤
│                    │Validation blocked: JSON body is invalid.                │
│                    │Fix the text before Save or Send.                        │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="05-raw"></a>

## Body: raw payload

JSON Structured/Text changes presentation only. JSON/Raw changes payload type: valid JSON converts to formatted text; Raw to JSON requires successful parsing and leaves the original intact on failure.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: Raw    Content-Type: application/xml                         │
│  > POST create          │1 <payment>                                                             │
│    ! broken             │2   <amount>1200</amount>                                               │
│                         │3   <currency>PLN</currency>                                            │
│▸ users                  │4 </payment>                                                            │
│                         │Text is preserved exactly when saved.                                   │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │No response yet.                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: Raw    Content-Type: application/xml          │
│  > POST create     │1 <payment>                                              │
│    ! broken        │2   <amount>1200</amount>                                │
│                    │3   <currency>PLN</currency>                             │
│▸ users             │4 </payment>                                             │
│                    │Text is preserved exactly when saved.                    │
│                    ├─────────────────────────────────────────────────────────┤
│                    │No response yet.                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="06-settings"></a>

## Request identity and effective settings

Settings shows effective collection/environment HTTP values without inventing request-local HTTP overrides. Name is editable; request path is not renamed here.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  Saved                                                           │
│▾ payments               │Params | Headers | Auth | Body | [Settings]                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Name: Create payment                                                    │
│  > POST create          │File: payments/create.yaml (read only)                                  │
│    ! broken             │Timeout: 30s (environment: test)                                        │
│                         │TLS certificate verification: enabled (collection)                      │
│▸ users                  │Request HTTP overrides: unavailable in MVP                              │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │Saved. Definition updated; response remains unchanged.                  │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  Saved                                            │
│▾ payments          │Params | Headers | Auth | Body | [Settings]              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Name: Create payment                                     │
│  > POST create     │File: payments/create.yaml (read only)                   │
│    ! broken        │Timeout: 30s (environment: test)                         │
│                    │TLS certificate verification: enabled (collection)       │
│▸ users             │Request HTTP overrides: unavailable in MVP               │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │Saved. Definition updated; response remains unchanged.   │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="07-auth-inherited"></a>

## Auth: inherited OAuth

Inherited fields are read only. Choosing OAuth2 creates a request override. None disables inherited auth. Token metadata belongs to the current effective configuration/environment.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | [Auth] | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Auth: [Inherit] | None | OAuth2                                         │
│  > POST create          │Effective source: collection Payments API                               │
│    ! broken             │Grant: client_credentials                                               │
│                         │Endpoint: {{oauth_token_url}}                                           │
│▸ users                  │Scopes: payments.write                                                  │
│                         │Token: ********  expires in 24m  [Show token]                           │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │No response yet.                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | [Auth] | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Auth: [Inherit] | None | OAuth2                          │
│  > POST create     │Effective source: collection Payments API                │
│    ! broken        │Grant: client_credentials                                │
│                    │Endpoint: {{oauth_token_url}}                            │
│▸ users             │Scopes: payments.write                                   │
│                    │Token: ********  expires in 24m  [Show token]            │
│                    ├─────────────────────────────────────────────────────────┤
│                    │No response yet.                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="08-auth-override"></a>

## Auth: request override

Credentials accept process environment references. Literal client secrets are rejected. Switching to Inherit removes the request override; undo can restore it.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | [Auth] | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Auth: Inherit | None | [OAuth2]                                         │
│  > POST create          │Grant: client_credentials                                               │
│    ! broken             │Endpoint: {{oauth_token_url}}                                           │
│                         │Client ID: ${PAYMENTS_CLIENT_ID}                                        │
│▸ users                  │Client secret: ${PAYMENTS_CLIENT_SECRET}                                │
│                         │Scopes: payments.write                                                  │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │Token: not acquired. Acquired on Send.                                  │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | [Auth] | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Auth: Inherit | None | [OAuth2]                          │
│  > POST create     │Grant: client_credentials                                │
│    ! broken        │Endpoint: {{oauth_token_url}}                            │
│                    │Client ID: ${PAYMENTS_CLIENT_ID}                         │
│▸ users             │Client secret: ${PAYMENTS_CLIENT_SECRET}                 │
│                    │Scopes: payments.write                                   │
│                    ├─────────────────────────────────────────────────────────┤
│                    │Token: not acquired. Acquired on Send.                   │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="09-auth-reveal"></a>

## Auth: explicit token reveal

Synthetic example only. Remask on leaving Auth, changing request/environment/effective auth, token replacement, or closing the session. No automatic clipboard action.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | [Auth] | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Auth: [Inherit] | None | OAuth2                                         │
│  > POST create          │Token visibility: revealed for this view                                │
│    ! broken             │Token: demo-opaque-token-not-a-real-credential                          │
│                         │[Hide token]                                                            │
│▸ users                  │Expires: 2026-09-30 12:30 UTC                                           │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │Token display never changes cache, logs or history.                     │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | [Auth] | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Auth: [Inherit] | None | OAuth2                          │
│  > POST create     │Token visibility: revealed for this view                 │
│    ! broken        │Token: demo-opaque-token-not-a-real-credential           │
│                    │[Hide token]                                             │
│▸ users             │Expires: 2026-09-30 12:30 UTC                            │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │Token display never changes cache, logs or history.      │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="10-sending"></a>

## Execution: sending current draft

Freeze a draft and environment snapshot for this send. Disable Send and mutating controls until completion; keep response scrolling and cancellation available.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Sending... / Cancel]  [Save] * Unsaved                                 │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | Text                               │
│  > POST create          │  amount     number   1200                                              │
│    ! broken             │  currency   string   PLN                                               │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │Sending draft... elapsed 1.2s                                           │
│                         │POST https://test.example.com/payments                                  │
│                         │[Cancel request]                                                        │
│                         │Any previous response is retained as cached data.                       │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Sending... / Cancel]  [Save] * Unsaved                  │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | Text                │
│  > POST create     │  amount     number   1200                               │
│    ! broken        │  currency   string   PLN                                │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │Sending draft... elapsed 1.2s                            │
│                    │POST https://test.example.com/payments                   │
│                    │[Cancel request]                                         │
│                    │Any previous response is retained as cached data.        │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="11-response"></a>

## Response: completed JSON exchange

The result is linked to the sent draft snapshot. Raw view shows response text; headers are redacted. Invalid JSON falls back to raw text with a parsing notice.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | Text                               │
│  > POST create          │  amount     number   1200                                              │
│    ! broken             │  currency   string   PLN                                               │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │201 Created | 184ms | 48 B | test | Live                                │
│                         │Received: 2026-09-30 12:06:00 UTC                                       │
│                         │[Body] Headers | Pretty JSON / Raw                                      │
│                         │{ "id": "pay_123", "status": "created" }                                │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | Text                │
│  > POST create     │  amount     number   1200                               │
│    ! broken        │  currency   string   PLN                                │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │201 Created | 184ms | 48 B | test | Live                 │
│                    │Received: 2026-09-30 12:06:00 UTC                        │
│                    │[Body] Headers | Pretty JSON / Raw                       │
│                    │{ "id": "pay_123", "status": "created" }                 │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="12-http-error"></a>

## Response: completed HTTP error

3xx/4xx/5xx stay responses. No automatic retries of mutation requests.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  Saved                                                           │
│▾ payments               │Params | [Headers] | Auth | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Name: Create payment                                                    │
│  > POST create          │Accept: application/json                                                │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │401 Unauthorized | 92ms | 36 B | test | Live                            │
│                         │[Body] Headers | Pretty JSON / Raw                                      │
│                         │{ "error": "insufficient_scope" }                                       │
│                         │HTTP exchange completed. Inspect body and auth.                         │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  Saved                                            │
│▾ payments          │Params | [Headers] | Auth | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Name: Create payment                                     │
│  > POST create     │Accept: application/json                                 │
│    ! broken        │                                                         │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │401 Unauthorized | 92ms | 36 B | test | Live             │
│                    │[Body] Headers | Pretty JSON / Raw                       │
│                    │{ "error": "insufficient_scope" }                        │
│                    │HTTP exchange completed. Inspect body and auth.          │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="13-cached"></a>

## Response: cached result

On environment change load only that environment cache; show No response if absent. On edit label any previous result as a previous execution, never a result of the current draft.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │[Params] | Headers | Auth | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Name: Create payment *                                                  │
│  > POST create          │Query: dry_run=false                                                    │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │201 Created | 184ms | 48 B | test | Cached                              │
│                         │Received: 2026-09-30 12:06:00 UTC                                       │
│                         │Previous execution; current draft may differ.                           │
│                         │{ "id": "pay_123", "status": "created" }                                │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │[Params] | Headers | Auth | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Name: Create payment *                                   │
│  > POST create     │Query: dry_run=false                                     │
│    ! broken        │                                                         │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │201 Created | 184ms | 48 B | test | Cached               │
│                    │Received: 2026-09-30 12:06:00 UTC                        │
│                    │Previous execution; current draft may differ.            │
│                    │{ "id": "pay_123", "status": "created" }                 │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="14-empty-body"></a>

## Response: no content

Empty payload is an explicit result state, not a viewer error.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  Saved                                                           │
│▾ payments               │Params | [Headers] | Auth | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Name: Create payment                                                    │
│  > POST create          │Accept: application/json                                                │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │204 No Content | 73ms | 0 B | test | Live                               │
│                         │[Body] Headers                                                          │
│                         │This response has no body.                                              │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  Saved                                            │
│▾ payments          │Params | [Headers] | Auth | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Name: Create payment                                     │
│  > POST create     │Accept: application/json                                 │
│    ! broken        │                                                         │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │204 No Content | 73ms | 0 B | test | Live                │
│                    │[Body] Headers                                           │
│                    │This response has no body.                               │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="15-validation"></a>

## Diagnostics: missing process variable

Diagnostics list all independent safe errors, offer Go to field where editable, and never print the secret value. Structural errors block Save; resolution errors block Send.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | [Auth] | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Auth: Inherit | None | [OAuth2]                                         │
│  > POST create          │Client secret: ${PAYMENTS_CLIENT_SECRET}                                │
│    ! broken             │! Process variable is unavailable                                       │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │[Diagnostics] Response Request Log                                      │
│                         │Stage: resolution | missing_process_variable                            │
│                         │payments/create.yaml > auth.client_secret                               │
│                         │Set PAYMENTS_CLIENT_SECRET before starting apitool.                     │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | [Auth] | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Auth: Inherit | None | [OAuth2]                          │
│  > POST create     │Client secret: ${PAYMENTS_CLIENT_SECRET}                 │
│    ! broken        │! Process variable is unavailable                        │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │[Diagnostics] Response Request Log                       │
│                    │Stage: resolution | missing_process_variable             │
│                    │payments/create.yaml > auth.client_secret                │
│                    │Set PAYMENTS_CLIENT_SECRET before starting apitool.      │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="16-transport"></a>

## Diagnostics: transport timeout

Timeout/cancellation cannot guarantee server rollback. Retry is an explicit Send action; keep the cached response separately labelled.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | Text                               │
│  > POST create          │  amount     number   1200                                              │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │[Diagnostics] Response Request Log                                      │
│                         │Stage: transport | timeout | elapsed 30s                                │
│                         │Host: test.example.com                                                  │
│                         │Check reachability and environment timeout.                             │
│                         │Outcome unknown: the server may have processed it.                      │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | Text                │
│  > POST create     │  amount     number   1200                               │
│    ! broken        │                                                         │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │[Diagnostics] Response Request Log                       │
│                    │Stage: transport | timeout | elapsed 30s                 │
│                    │Host: test.example.com                                   │
│                    │Check reachability and environment timeout.              │
│                    │Outcome unknown: the server may have processed it.       │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="17-log"></a>

## Diagnostics: redacted request log

OAuth failure points to auth configuration without exposing returned secret content. Log safe stage metadata; do not include raw request bodies.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | [Auth] | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Auth: [Inherit] | None | OAuth2                                         │
│  > POST create          │Token: ********  [Show token]                                           │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │Response Diagnostics [Request Log]                                      │
│                         │12:06:00 resolution: complete                                           │
│                         │12:06:00 oauth: invalid_client                                          │
│                         │Host: auth.example.com | grant: client_credentials                      │
│                         │Client secret: [redacted] | token: [redacted]                           │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | [Auth] | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Auth: [Inherit] | None | OAuth2                          │
│  > POST create     │Token: ********  [Show token]                            │
│    ! broken        │                                                         │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │Response Diagnostics [Request Log]                       │
│                    │12:06:00 resolution: complete                            │
│                    │12:06:00 oauth: invalid_client                           │
│                    │Host: auth.example.com | grant: client_credentials       │
│                    │Client secret: [redacted] | token: [redacted]            │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="18-invalid-definition"></a>

## Diagnostics: invalid selected definition

Never show a sibling request in the editor while an invalid item is selected. Parsed field errors can use typed repair; malformed YAML requires external repair in this proposed MVP recovery route.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ Invalid: payments/broken.yaml                                         │
│                         │[Send disabled]  Saved                                                  │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Selected: payments/broken.yaml                                          │
│    POST create          │! Definition cannot be loaded                                           │
│  > ! broken             │Request controls unavailable                                            │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │[Diagnostics] Response Request Log                                      │
│                         │Stage: load | invalid_yaml                                              │
│                         │payments/broken.yaml: line 4                                            │
│                         │Repair this file externally, then [Reload].                             │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ Invalid: payments/broken.yaml                          │
│                    │[Send disabled]  Saved                                   │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Selected: payments/broken.yaml                           │
│    POST create     │! Definition cannot be loaded                            │
│  > ! broken        │Request controls unavailable                             │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │[Diagnostics] Response Request Log                       │
│                    │Stage: load | invalid_yaml                               │
│                    │payments/broken.yaml: line 4                             │
│                    │Repair this file externally, then [Reload].              │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="19-dirty-navigation"></a>

## Unsaved changes: navigation

Also applies to collection/environment switching and quit. Cancel restores the exact selection, focus and scroll. Save validates then continues only on success.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | Text                               │
│  > POST create          │  amount     number   1200                                              │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                 ┌──────────────────────────────────────────────────────────────┐                 │
│                 │ Unsaved changes                                              │─────────────────┤
│                 │ payments/create.yaml                                         │                 │
│                 │ Continue to payments/list?                                   │                 │
│                 │ [Save and continue] [Discard] [Cancel*]                      │                 │
│                 └──────────────────────────────────────────────────────────────┘                 │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | Text                │
│  > POST create     │  amount     number   1200                               │
│    ! broken        │                                                         │
│                    │                                                         │
│▸ users┌──────────────────────────────────────────────────────────────┐       │
│       │ Unsaved changes                                              │       │
│       │ payments/create.yaml                                         │───────┤
│       │ Continue to payments/list?                                   │       │
│       │ [Save and continue] [Discard] [Cancel*]                      │       │
│       └──────────────────────────────────────────────────────────────┘       │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="20-save-failure"></a>

## Unsaved changes: save failed

Retain the draft and pending destination. Never navigate after a failed save. Return to editing dismisses the dialog without changing the file.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | Text                               │
│  > POST create          │  amount     number   1200                                              │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                 ┌──────────────────────────────────────────────────────────────┐                 │
│                 │ Save failed                                                  │─────────────────┤
│                 │ payments/create.yaml                                         │                 │
│                 │ Permission denied. Your draft is retained.                   │                 │
│                 │ [Retry save] [Return to editing*]                            │                 │
│                 └──────────────────────────────────────────────────────────────┘                 │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | Text                │
│  > POST create     │  amount     number   1200                               │
│    ! broken        │                                                         │
│                    │                                                         │
│▸ users┌──────────────────────────────────────────────────────────────┐       │
│       │ Save failed                                                  │       │
│       │ payments/create.yaml                                         │───────┤
│       │ Permission denied. Your draft is retained.                   │       │
│       │ [Retry save] [Return to editing*]                            │       │
│       └──────────────────────────────────────────────────────────────┘       │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="21-confirm-send"></a>

## Dangerous send: explicit confirmation

Only when --confirm-dangerous is enabled and method is POST/PUT/PATCH/DELETE. Resolve and validate before opening. Confirm sends the exact frozen draft once; Cancel sends nothing.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | Text                               │
│  > POST create          │  amount     number   1200                                              │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                 ┌──────────────────────────────────────────────────────────────┐                 │
│                 │ Confirm request                                              │                 │
│                 │ Environment: test | POST                                     │─────────────────┤
│                 │ https://test.example.com/payments                            │                 │
│                 │ Unsaved draft; sending does not save it.                     │                 │
│                 │ [Cancel*] [Send request]                                     │                 │
│                 └──────────────────────────────────────────────────────────────┘                 │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | Text                │
│  > POST create     │  amount     number   1200                               │
│    ! broken        │                                                         │
│       ┌──────────────────────────────────────────────────────────────┐       │
│▸ users│ Confirm request                                              │       │
│       │ Environment: test | POST                                     │       │
│       │ https://test.example.com/payments                            │───────┤
│       │ Unsaved draft; sending does not save it.                     │       │
│       │ [Cancel*] [Send request]                                     │       │
│       └──────────────────────────────────────────────────────────────┘       │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

Dialog actions marked with `*` have initial focus. All times and token examples are synthetic.
