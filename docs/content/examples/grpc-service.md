---
title: "gRPC Service Example"
weight: 2
---

# Complete gRPC Service Example

A complete example of a gRPC service definition for an order management system.

## order_service.proto

```proto
syntax = "proto3";

package order.v1;

option go_package = "github.com/example/order/v1;orderv1";
option java_package = "com.example.order.v1";
option java_multiple_files = true;

import "google/protobuf/timestamp.proto";

// Order status enumeration
enum OrderStatus {
  ORDER_STATUS_UNSPECIFIED = 0;
  ORDER_STATUS_PENDING = 1;
  ORDER_STATUS_CONFIRMED = 2;
  ORDER_STATUS_SHIPPED = 3;
  ORDER_STATUS_DELIVERED = 4;
  ORDER_STATUS_CANCELLED = 5;
}

// Payment method enumeration
enum PaymentMethod {
  PAYMENT_METHOD_UNSPECIFIED = 0;
  PAYMENT_METHOD_CREDIT_CARD = 1;
  PAYMENT_METHOD_DEBIT_CARD = 2;
  PAYMENT_METHOD_PAYPAL = 3;
  PAYMENT_METHOD_BANK_TRANSFER = 4;
}

// Money representation
message Money {
  // Currency code (ISO 4217)
  string currency_code = 1;

  // Whole units of the amount
  int64 units = 2;

  // Number of nano (10^-9) units
  int32 nanos = 3;
}

// Address information
message Address {
  string street = 1;
  string city = 2;
  string state = 3;
  string postal_code = 4;
  string country = 5;
}

// Order item
message OrderItem {
  string product_id = 1;
  string product_name = 2;
  int32 quantity = 3;
  Money unit_price = 4;
  Money total_price = 5;
}

// Order entity
message Order {
  // Unique order identifier
  string id = 1;

  // Customer information
  string customer_id = 2;
  string customer_email = 3;

  // Order items
  repeated OrderItem items = 4;

  // Pricing
  Money subtotal = 5;
  Money tax = 6;
  Money shipping_cost = 7;
  Money total = 8;

  // Payment
  PaymentMethod payment_method = 9;

  // Shipping
  Address shipping_address = 10;

  // Status
  OrderStatus status = 11;

  // Timestamps
  google.protobuf.Timestamp created_at = 12;
  google.protobuf.Timestamp updated_at = 13;

  // Optional tracking number
  string tracking_number = 14;

  // Internal notes
  string notes = 15;
}

// Create order request
message CreateOrderRequest {
  string customer_id = 1;
  string customer_email = 2;
  repeated OrderItem items = 3;
  PaymentMethod payment_method = 4;
  Address shipping_address = 5;
  string notes = 6;
}

// Create order response
message CreateOrderResponse {
  Order order = 1;
}

// Get order request
message GetOrderRequest {
  string id = 1;
}

// Get order response
message GetOrderResponse {
  Order order = 1;
  bool found = 2;
}

// List orders request
message ListOrdersRequest {
  // Pagination
  int32 page = 1;
  int32 page_size = 2;

  // Filters
  string customer_id = 3;
  OrderStatus status = 4;

  // Date range
  google.protobuf.Timestamp start_date = 5;
  google.protobuf.Timestamp end_date = 6;
}

// List orders response
message ListOrdersResponse {
  repeated Order orders = 1;
  int32 total_count = 2;
  int32 page = 3;
  int32 page_size = 4;
}

// Update order status request
message UpdateOrderStatusRequest {
  string id = 1;
  OrderStatus status = 2;
  string tracking_number = 3;
}

// Update order status response
message UpdateOrderStatusResponse {
  Order order = 1;
}

// Cancel order request
message CancelOrderRequest {
  string id = 1;
  string reason = 2;
}

// Cancel order response
message CancelOrderResponse {
  bool success = 1;
  string message = 2;
}

// Order statistics request
message GetOrderStatsRequest {
  google.protobuf.Timestamp start_date = 1;
  google.protobuf.Timestamp end_date = 2;
}

// Order statistics response
message GetOrderStatsResponse {
  int32 total_orders = 1;
  Money total_revenue = 2;
  int32 pending_orders = 3;
  int32 delivered_orders = 4;
  int32 cancelled_orders = 5;
}

// OrderService manages customer orders
service OrderService {
  // CreateOrder creates a new order
  rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse);

  // GetOrder retrieves an order by ID
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);

  // ListOrders lists orders with filters and pagination
  rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);

  // UpdateOrderStatus updates the status of an order
  rpc UpdateOrderStatus(UpdateOrderStatusRequest) returns (UpdateOrderStatusResponse);

  // CancelOrder cancels an order
  rpc CancelOrder(CancelOrderRequest) returns (CancelOrderResponse);

  // GetOrderStats returns order statistics
  rpc GetOrderStats(GetOrderStatsRequest) returns (GetOrderStatsResponse);

  // StreamOrderUpdates streams real-time order updates
  rpc StreamOrderUpdates(stream GetOrderRequest) returns (stream Order);
}
```

## Push to Spoke

```bash
spoke-cli push \
  -module order-service \
  -version v1.0.0 \
  -dir ./proto \
  -registry http://localhost:8080 \
  -description "Order management service"
```

## Compile for Different Languages

### Go

```bash
spoke-cli pull -module order-service -version v1.0.0 -dir ./proto -recursive
spoke-cli compile -dir ./proto -out ./generated/go -lang go
```

Generated files:
- `order_service.pb.go` - Message definitions
- `order_service_grpc.pb.go` - gRPC service stubs

### Python

```bash
spoke-cli compile -dir ./proto -out ./generated/python -lang python
```

Generated files:
- `order_service_pb2.py` - Message definitions
- `order_service_pb2_grpc.py` - gRPC service stubs

### Java

```bash
spoke-cli compile -dir ./proto -out ./generated/java -lang java
```

Generated files in `com/example/order/v1/`:
- `OrderServiceProto.java` - Message definitions
- `OrderServiceGrpc.java` - gRPC service stubs

## Usage Examples

### Go Server Implementation

```go
package main

import (
    "context"
    "fmt"
    "time"

    pb "github.com/example/order/v1"
    "google.golang.org/protobuf/types/known/timestamppb"
)

type orderServer struct {
    pb.UnimplementedOrderServiceServer
    orders map[string]*pb.Order
}

func (s *orderServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
    order := &pb.Order{
        Id:              fmt.Sprintf("order-%d", time.Now().Unix()),
        CustomerId:      req.CustomerId,
        CustomerEmail:   req.CustomerEmail,
        Items:           req.Items,
        PaymentMethod:   req.PaymentMethod,
        ShippingAddress: req.ShippingAddress,
        Status:          pb.OrderStatus_ORDER_STATUS_PENDING,
        CreatedAt:       timestamppb.Now(),
        UpdatedAt:       timestamppb.Now(),
        Notes:           req.Notes,
    }

    // Calculate totals
    subtotal := &pb.Money{CurrencyCode: "USD", Units: 0, Nanos: 0}
    for _, item := range order.Items {
        subtotal.Units += item.TotalPrice.Units
        subtotal.Nanos += item.TotalPrice.Nanos
    }

    order.Subtotal = subtotal
    order.Tax = &pb.Money{CurrencyCode: "USD", Units: subtotal.Units / 10, Nanos: 0}
    order.ShippingCost = &pb.Money{CurrencyCode: "USD", Units: 10, Nanos: 0}
    order.Total = &pb.Money{
        CurrencyCode: "USD",
        Units:        subtotal.Units + order.Tax.Units + order.ShippingCost.Units,
        Nanos:        0,
    }

    s.orders[order.Id] = order

    return &pb.CreateOrderResponse{Order: order}, nil
}

func (s *orderServer) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
    order, found := s.orders[req.Id]
    return &pb.GetOrderResponse{
        Order: order,
        Found: found,
    }, nil
}
```

### Python Client Implementation

```python
import grpc
import order_service_pb2
import order_service_pb2_grpc
from google.protobuf import timestamp_pb2

def create_order():
    channel = grpc.insecure_channel('localhost:50051')
    stub = order_service_pb2_grpc.OrderServiceStub(channel)

    # Create order items
    items = [
        order_service_pb2.OrderItem(
            product_id="prod-1",
            product_name="Widget",
            quantity=2,
            unit_price=order_service_pb2.Money(
                currency_code="USD",
                units=25,
                nanos=0
            ),
            total_price=order_service_pb2.Money(
                currency_code="USD",
                units=50,
                nanos=0
            )
        )
    ]

    # Create shipping address
    address = order_service_pb2.Address(
        street="123 Main St",
        city="San Francisco",
        state="CA",
        postal_code="94105",
        country="USA"
    )

    # Create order
    request = order_service_pb2.CreateOrderRequest(
        customer_id="cust-123",
        customer_email="customer@example.com",
        items=items,
        payment_method=order_service_pb2.PAYMENT_METHOD_CREDIT_CARD,
        shipping_address=address
    )

    response = stub.CreateOrder(request)
    print(f"Created order: {response.order.id}")
    print(f"Total: ${response.order.total.units}.{response.order.total.nanos:02d}")

    return response.order

if __name__ == '__main__':
    order = create_order()
```

## Key Features Demonstrated

1. **Enums**: `OrderStatus`, `PaymentMethod`
2. **Nested Messages**: `OrderItem`, `Address`, `Money`
3. **Repeated Fields**: `items` in `Order`
4. **Timestamp**: Using `google.protobuf.Timestamp`
5. **Pagination**: `page` and `page_size` in `ListOrdersRequest`
6. **Filtering**: Status and date range filters
7. **Streaming**: `StreamOrderUpdates` RPC
8. **Documentation**: Inline comments for all fields
9. **Multi-language**: Java package options

## Best Practices

1. **Use standard types**: `google.protobuf.Timestamp` for timestamps
2. **Enums start at 0**: `UNSPECIFIED` value for unknown/default
3. **Consistent naming**: `snake_case` for fields, `PascalCase` for messages
4. **Document everything**: Comments for all messages and fields
5. **Versioning**: Package name includes version (`order.v1`)
6. **Language options**: Set package options for target languages

## Next Steps

- [Go Service Example](/examples/go-service/) - Full server implementation
- [Python Client Example](/examples/python-client/) - Complete client
- [Microservices Example](/examples/microservices/) - Multi-service architecture
