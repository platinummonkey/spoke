---
title: "Request Playground"
weight: 3
---

# Request/Response Playground

The Try It Out playground lets you build, validate, and test JSON requests against your protobuf schemas without writing any code.

## Overview

The playground provides:
- **Auto-Generated Samples**: Instant request templates from schemas
- **JSON Editor**: Simple textarea-based editor
- **Real-Time Validation**: Catch errors before sending requests
- **Response Viewer**: Formatted response display
- **Copy to Clipboard**: Easy code integration

## Accessing the Playground

1. Navigate to **API Explorer** tab
2. Expand a service and select a method
3. Click the **Try It Out** tab (2nd tab in method detail)
4. The playground appears with a pre-generated sample request

## Request Builder

### Auto-Generated Sample

When you open the playground, a sample JSON request is automatically generated based on the message schema:

```json
{
  "email": "user@example.com",
  "name": "John Doe",
  "phone": "+1-555-0123"
}
```

**Sample Value Rules:**
- Fields are populated with realistic test data
- Values match field name patterns (email → email format)
- Required fields are always included
- Optional fields are shown but can be removed
- Repeated fields shown as arrays with one example item

### Editing Requests

The JSON editor is a simple textarea where you can:
- Modify field values
- Add or remove optional fields
- Test different data combinations
- Format JSON as you prefer (spacing doesn't matter)

**Example Edits:**

```json
{
  "email": "alice@company.com",
  "name": "Alice Smith",
  "phone": "+1-555-9999",
  "role": "ADMIN"
}
```

### Generate Sample Button

Click **"Generate Sample"** to:
- Reset to default values
- Restore original field structure
- Undo all modifications
- Start fresh with a new template

## JSON Validation

The playground validates your JSON in real-time against the protobuf schema.

### Validation Rules

**Type Checking:**
```json
{
  "age": "not a number"  // ❌ Error: must be int32
}
```

**Required Fields:**
```json
{
  "name": "John Doe"
  // ❌ Error: missing required field "email"
}
```

**Enum Values:**
```json
{
  "status": "INVALID_STATUS"  // ❌ Error: not a valid enum value
}
```

**Repeated Fields:**
```json
{
  "tags": "single-value"  // ❌ Error: must be an array
}
```

**Nested Messages:**
```json
{
  "address": "not an object"  // ❌ Error: must be a message
}
```

### Validation Errors

Errors appear below the JSON editor in red with:
- **Field path**: Which field has the error
- **Error type**: What's wrong
- **Expected type**: What was expected

**Example Error Display:**

```
⚠ Validation Errors:
• email: Invalid email format
• age: Expected int32, got string
• status: Invalid enum value "PENDING"
```

### Valid JSON Indicators

When JSON is valid:
- ✓ Green checkmark appears
- No error messages shown
- "Send Request" button enabled (if simulation available)

## Field Type Reference

### String Fields

```json
{
  "name": "John Doe",
  "description": "User profile"
}
```

- Must be quoted
- Can be empty: `""`
- Special characters need escaping: `\n`, `\t`, `\"`

### Numeric Fields

```json
{
  "age": 25,
  "balance": 1234.56,
  "count": 0
}
```

- No quotes
- Decimals for float/double
- Integers for int32/int64

### Boolean Fields

```json
{
  "active": true,
  "deleted": false
}
```

- Lowercase only: `true` or `false`
- No quotes

### Enum Fields

```json
{
  "status": "ACTIVE"
}
```

- Must match enum value exactly (case-sensitive)
- Check schema for valid values
- Usually UPPER_SNAKE_CASE

### Repeated Fields (Arrays)

```json
{
  "tags": ["tag1", "tag2", "tag3"],
  "scores": [95, 87, 92]
}
```

- Use square brackets `[]`
- Can be empty: `[]`
- All elements must be same type

### Nested Messages

```json
{
  "address": {
    "street": "123 Main St",
    "city": "Springfield",
    "state": "IL",
    "zip": "62701"
  }
}
```

- Use curly braces `{}`
- Follows same validation rules
- Can nest multiple levels

### Timestamp Fields

```json
{
  "created_at": {
    "seconds": 1706169600,
    "nanos": 0
  }
}
```

- Protobuf Timestamp is a message with `seconds` and `nanos`
- Seconds: Unix timestamp (int64)
- Nanos: Nanosecond precision (int32, 0-999999999)

### Optional vs Required

**Required field (proto3 implicit):**
```json
{
  "user_id": "123"  // Must be present
}
```

**Optional field:**
```json
{
  "phone": "+1-555-0123"  // Can be omitted
}
// or
{}  // Valid if phone is optional
```

## Response Viewer

The response section shows:
- **Success state**: Green checkmark with formatted response
- **Error state**: Red error icon with error message
- **Copy button**: Copy response JSON to clipboard

**Mock Response Example:**

```json
{
  "user_id": "usr_abc123",
  "email": "user@example.com",
  "name": "John Doe",
  "created_at": {
    "seconds": 1706169600,
    "nanos": 0
  },
  "status": "ACTIVE"
}
```

Note: Current implementation shows mock responses for demonstration. Future versions may support live API testing.

## Copy to Clipboard

### Copying Request JSON

1. Click the **Copy Request** button
2. JSON is copied to clipboard
3. Paste into your code, Postman, curl, etc.

**Example use in curl:**

```bash
curl -X POST http://localhost:50051/UserService/CreateUser \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "name": "John Doe",
    "phone": "+1-555-0123"
  }'
```

### Copying Response JSON

1. Click the **Copy Response** button
2. Response JSON copied to clipboard
3. Use for testing, documentation, or mocking

## Common Workflows

### Testing Field Validation

**Goal**: Verify email validation

1. Generate sample request
2. Change email to invalid format: `"not-an-email"`
3. See validation error
4. Correct to valid format: `"user@example.com"`
5. Validation passes

### Building Complex Requests

**Goal**: Create nested request with address

1. View request schema to understand structure
2. Edit JSON to add address:
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "address": {
    "street": "123 Main St",
    "city": "Springfield",
    "state": "IL",
    "zip": "62701"
  }
}
```
3. Validate structure
4. Copy to use in code

### Testing Optional Fields

**Goal**: See what happens without optional fields

1. Generate sample (includes all fields)
2. Remove optional fields:
```json
{
  "email": "user@example.com",
  "name": "John Doe"
  // Removed: phone (optional)
}
```
3. Verify still validates
4. Test with and without optional fields

### Exploring Enum Values

**Goal**: Understand valid status values

1. View message schema to see enum values
2. Try each enum value in JSON:
```json
{
  "status": "ACTIVE"
}
// Then try: "INACTIVE", "DELETED", etc.
```
3. Invalid enum shows error
4. Document valid values

## Integration with Code Examples

After building a request in the playground:

1. Click **Usage Examples** tab
2. Select your language
3. Replace sample values with your tested JSON

**Example (Python):**

From playground:
```json
{
  "email": "alice@company.com",
  "name": "Alice Smith",
  "role": "ADMIN"
}
```

In code example:
```python
response = client.CreateUser(
    CreateUserRequest(
        email="alice@company.com",
        name="Alice Smith",
        role=Role.ADMIN
    )
)
```

## Tips & Best Practices

### Efficient Testing

- **Start simple**: Test with minimal required fields first
- **Add incrementally**: Add optional fields one at a time
- **Save examples**: Keep working JSON samples for reuse
- **Test edge cases**: Empty strings, zero values, maximum values

### Understanding Schemas

- **Read schema first**: View message schema before editing
- **Check field types**: Match JSON types to proto types
- **Note requirements**: Identify required vs optional fields
- **Explore enums**: List valid enum values before testing

### Validation Strategy

- **Fix one error at a time**: Don't overwhelm yourself
- **Use sample generator**: Reset when lost
- **Copy valid JSON**: Build a library of working examples
- **Test variations**: Try different combinations

## Keyboard Shortcuts

- **CMD+A / CTRL+A**: Select all in editor
- **CMD+C / CTRL+C**: Copy selected text
- **CMD+V / CTRL+V**: Paste text
- **Tab**: Indent (if supported)

## Limitations

Current limitations:
- **No live API calls**: Mock responses only (currently)
- **No syntax highlighting**: Plain textarea (for security)
- **Basic formatting**: Manual JSON formatting required
- **No auto-complete**: Type field names manually

## Troubleshooting

### Validation not working

**Symptoms**: No errors shown for invalid JSON
**Possible causes**:
- JSON syntax error (malformed)
- Schema not loaded

**Fix**: Check JSON syntax with a validator first

### Can't edit JSON

**Symptoms**: Editor is read-only or not responding
**Fix**: Refresh page, ensure JavaScript is enabled

### Sample not generating

**Symptoms**: "Generate Sample" doesn't work
**Fix**: Ensure message schema loaded correctly

### Unexpected validation errors

**Symptoms**: Valid-looking JSON fails validation
**Check**:
- Field names match schema exactly (case-sensitive)
- Enum values are exact matches
- Types match (string vs number)
- Nested structure matches schema

## Future Enhancements

Planned features:
- Live API testing (send real requests)
- Syntax highlighting in JSON editor
- Auto-completion for field names
- Schema-aware editor with inline validation
- Save/load request templates
- Request history
- Authentication configuration

## What's Next?

- [**Code Examples**](code-examples) - Convert validated JSON to client code
- [**API Explorer**](api-explorer) - Explore more methods
- [**CLI Testing**](../cli-reference#testing) - Test with command-line tools
- [**gRPC Testing Tools**](../../examples/testing) - Production testing strategies

## Related Documentation

- [Proto JSON Mapping](https://protobuf.dev/programming-guides/proto3/#json) - Official JSON encoding guide
- [gRPC Error Codes](https://grpc.io/docs/guides/error/) - Understanding errors
- [Postman gRPC](https://learning.postman.com/docs/sending-requests/grpc/grpc-request-interface/) - Testing with Postman
