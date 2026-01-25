---
title: "API Explorer"
weight: 1
---

# Interactive API Explorer

The API Explorer provides an interactive interface for browsing protobuf services, exploring methods, and understanding message schemas without writing any code.

## Overview

The API Explorer tab (4th tab in Module Detail) offers:
- **Service Browser**: Accordion-style list of all services
- **Method Viewer**: Detailed view of each method's signature
- **Schema Inspector**: Interactive message and field exploration
- **Type Visualization**: Color-coded type badges for quick identification

## Accessing the API Explorer

1. Navigate to any module from the home page
2. Select a version from the dropdown
3. Click the **API Explorer** tab
4. Browse services and click to expand

## Service Browser

### Viewing Services

The service browser displays all services defined in your proto files:

```
UserService (3 methods)
  ├── CreateUser
  ├── GetUser
  └── UpdateUser

OrderService (5 methods)
  ├── CreateOrder
  ├── GetOrder
  ├── ListOrders
  ├── UpdateOrder
  └── CancelOrder
```

**Features:**
- Accordion-style expandable list
- Method count badge for each service
- Click to expand/collapse
- Smooth animations

### Service Card

Each service shows:
- **Service name**: Fully qualified proto service name
- **Method count**: Number of RPC methods defined
- **Expand indicator**: Arrow icon shows expand/collapse state

## Method Details

Click any method to view its detailed signature and schemas.

### Method Viewer Layout

The method detail view has two tabs:

#### 1. Message Schema Tab

Shows side-by-side view of:
- **Left side**: Request message schema
- **Right side**: Response message schema

Both sides display:
- Message name
- Field list with full details
- Nested message support (expandable)

#### 2. Try It Out Tab

Interactive playground for building requests:
- Sample JSON generator
- JSON editor with validation
- Response viewer
- Copy to clipboard

[Learn more about the playground →](playground)

### Method Signature

The header displays:
- **Method name**: e.g., `CreateUser`
- **Streaming type**: Unary, Client Stream, Server Stream, or Bidirectional
- **Request type**: Input message name
- **Response type**: Output message name

**Streaming Indicators:**

| Type | Icon | Description |
|------|------|-------------|
| Unary | `→` | Single request, single response |
| Client Stream | `⇉ →` | Stream of requests, single response |
| Server Stream | `→ ⇉` | Single request, stream of responses |
| Bidirectional | `⇄` | Stream of requests and responses |

## Message Schema Inspector

### Field Display

Each field in a message shows:

```
┌─────────────────────────────────────────────┐
│ Field Name          Type         Number      │
│ user_id             string       1 [required]│
│ email               string       2 [required]│
│ phone_number        string       3 [optional]│
│ roles               Role         4 [repeated]│
│ created_at          Timestamp    5 [optional]│
└─────────────────────────────────────────────┘
```

**Field Components:**
- **Name**: Field identifier (snake_case from proto)
- **Type**: Protobuf type with color badge
- **Number**: Field number from proto definition
- **Label**: Required, optional, or repeated

### Type Color Coding

Fields are color-coded by type for quick scanning:

| Color | Types |
|-------|-------|
| 🟣 Purple | `string`, `bytes` |
| 🔵 Blue | `int32`, `int64`, `uint32`, `uint64`, `sint32`, `sint64`, `fixed32`, `fixed64`, `sfixed32`, `sfixed64`, `float`, `double` |
| 🟢 Green | `bool` |
| 🟠 Orange | Message references (nested types) |
| 🔷 Teal | Enum types |
| ⚪ Gray | Other types |

### Nested Messages

When a field references another message type:
1. Click the message type badge (orange)
2. Message expands inline to show nested fields
3. Click again to collapse
4. Supports multiple levels of nesting

**Example:**

```
ContactInfo (Message)
  ├── phone: string
  ├── email: string
  └── address: Address (Message)
      ├── street: string
      ├── city: string
      ├── state: string
      └── zip: string
```

### Enum Values

Enum fields display all possible values:

```
Status: enum
  ├── STATUS_UNKNOWN = 0
  ├── STATUS_ACTIVE = 1
  ├── STATUS_INACTIVE = 2
  └── STATUS_DELETED = 3
```

### Field Labels

**Required** (`required`):
- Field must be present in request
- Validation will fail if missing
- Proto3 implicit (non-optional, non-repeated)

**Optional** (`optional`):
- Field may be omitted
- Has default value if not provided
- Explicitly marked in proto

**Repeated** (`repeated`):
- Field is an array/list
- Can have 0 or more values
- Order is preserved

## Field Comments

If proto files include comments, they appear as tooltips:

```protobuf
// User's email address for login and notifications
string email = 2;
```

Hover over the field name to see the tooltip with the full comment.

## Example Exploration Flow

### Scenario: Exploring a User Service

1. **Open API Explorer tab** in the user-service module
2. **View services**: See `UserService` listed
3. **Expand service**: Click to see 3 methods
4. **Select CreateUser**: Click the method name
5. **View request schema**: See `CreateUserRequest` fields:
   - `email: string` (required)
   - `name: string` (required)
   - `phone_number: string` (optional)
6. **View response schema**: See `CreateUserResponse` fields:
   - `user_id: string` (required)
   - `created_at: Timestamp` (optional)
7. **Click Timestamp**: Expand to see nested fields:
   - `seconds: int64`
   - `nanos: int32`
8. **Switch to Try It Out**: Build a sample request

## Use Cases

### Understanding Service APIs

**Before writing code:**
1. Browse services to understand available operations
2. Review method signatures to plan integration
3. Explore message schemas to understand data structures
4. Identify required vs optional fields

### API Documentation Review

**As a technical writer:**
1. Navigate through all services systematically
2. Verify method names and descriptions
3. Check field naming consistency
4. Document any special requirements

### Schema Validation

**As a backend developer:**
1. Verify proto definitions match implementation
2. Check field numbers for consistency
3. Confirm enum values are correct
4. Validate nested message structures

### Client Integration Planning

**As a frontend developer:**
1. Identify which methods you need
2. Note required fields for validation
3. Check response structure for UI design
4. Copy field names for accurate mapping

## Tips & Best Practices

### Efficient Navigation

- **Collapse unused services**: Keep your view focused
- **Use browser search**: CMD+F within the page for specific terms
- **Bookmark deep links**: URL updates as you navigate
- **Open multiple tabs**: Compare different modules side-by-side

### Understanding Complex Schemas

- **Start at the top level**: Understand the service first
- **Expand incrementally**: Don't expand all nested messages at once
- **Follow references**: Click message types to see definitions
- **Check field numbers**: Useful for understanding wire format

### Keyboard Navigation

- **Tab**: Move between interactive elements
- **Enter**: Expand/collapse accordion items
- **Arrow keys**: Navigate within dropdowns
- **Escape**: Close modals and popovers

## Limitations

Current limitations to be aware of:
- **Proto comments**: Only displayed if present in proto files
- **Deprecated fields**: Not visually distinguished yet
- **Custom options**: Proto custom options not displayed
- **Field constraints**: Validation rules not shown (e.g., max length)

## Troubleshooting

### Service not appearing

**Possible causes:**
- Proto file syntax error
- Service defined in non-loaded file
- File not included in module version

**Solution:** Check proto file syntax and verify it's included in the pushed version.

### Message not expanding

**Possible causes:**
- Message type reference is invalid
- Circular reference in proto definitions

**Solution:** Verify proto definitions are valid with `protoc`.

### Types showing as "unknown"

**Possible causes:**
- Import path incorrect
- Dependent module not available
- Type defined in external package

**Solution:** Ensure all dependencies are pushed to Spoke registry.

## What's Next?

- [**Code Examples**](code-examples) - Generate client code for your language
- [**Try It Out Playground**](playground) - Build and test requests interactively
- [**Type Explorer**](../types) - View all messages and enums in one place
- [**Search**](search) - Find specific services, messages, or fields quickly

## Related Documentation

- [CLI Reference](../cli-reference) - Command-line tools for proto management
- [API Reference](../api-reference) - REST API endpoints
- [Proto Best Practices](../../examples/proto-style) - Writing maintainable protos
