# Cribb Backend API Documentation

## Handler Checklist

### Authentication Handlers
- [x] RegisterHandler
- [x] LoginHandler
- [x] GetUserProfileHandler

### User Handlers
- [x] GetUsersHandler
- [x] GetUserByUsernameHandler
- [x] GetUsersByScoreHandler

### Group Handlers
- [x] CreateGroupHandler
- [x] JoinGroupHandler
- [x] GetGroupMembersHandler

### Chore Handlers
- [x] CreateIndividualChoreHandler
- [x] CreateRecurringChoreHandler
- [x] GetUserChoresHandler
- [x] GetGroupChoresHandler
- [x] GetGroupRecurringChoresHandler
- [x] CompleteChoreHandler
- [x] UpdateChoreHandler
- [x] DeleteChoreHandler
- [x] UpdateRecurringChoreHandler
- [x] DeleteRecurringChoreHandler

### Pantry Handlers
- [x] AddPantryItemHandler
- [x] UsePantryItemHandler
- [x] GetPantryItemsHandler
- [x] DeletePantryItemHandler
- [x] GetPantryWarningsHandler
- [x] GetPantryExpiringHandler
- [x] GetPantryShoppingListHandler
- [x] GetPantryHistoryHandler
- [x] MarkNotificationReadHandler
- [x] DeleteNotificationHandler

### Shopping Cart Handlers
- [x] AddShoppingCartItemHandler
- [x] UpdateShoppingCartItemHandler
- [x] DeleteShoppingCartItemHandler
- [x] ListShoppingCartItemsHandler
- [x] GetShoppingCartActivityHandler
- [x] MarkActivityReadHandler

## API Details

### Authentication Endpoints

#### 1. RegisterHandler
**Endpoint:** `/api/register`  
**Method:** POST  
**Request Body:**
```json
{
  "username": "string",
  "password": "string",
  "name": "string",
  "phone_number": "string",
  "room_number": "string",
  "group": "string (optional)",
  "groupCode": "string (optional)"
}
```
**Models Used:**
- User
- Group

**Response:**
```json
{
  "success": true,
  "token": "JWT token string",
  "user": {
    "id": "string",
    "email": "string",
    "firstName": "string",
    "lastName": "string", 
    "phone": "string",
    "roomNo": "string",
    "groupCode": "string"
  },
  "message": "Registration successful"
}
```

#### 2. LoginHandler
**Endpoint:** `/api/login`  
**Method:** POST  
**Request Body:**
```json
{
  "username": "string",
  "password": "string"
}
```
**Models Used:**
- User

**Response:**
```json
{
  "success": true,
  "token": "JWT token string",
  "user": {
    "id": "string",
    "email": "string",
    "firstName": "string",
    "lastName": "string", 
    "phone": "string",
    "roomNo": "string",
    "groupCode": "string",
    "groupName": "string"
  },
  "message": "Login successful"
}
```

#### 3. GetUserProfileHandler
**Endpoint:** `/api/users/profile`  
**Method:** GET  
**Authentication:** Required (JWT Token)  
**Models Used:**
- User

**Response:**
```json
{
  "id": "string",
  "username": "string",
  "name": "string",
  "phone_number": "string",
  "room_number": "string",
  "score": number,
  "group": "string",
  "group_code": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

### User Endpoints

#### 4. GetUsersHandler
**Endpoint:** `/api/users`  
**Method:** GET  
**Authentication:** Required (JWT Token)  
**Models Used:**
- User

**Response:**
```json
[
  {
    "id": "string",
    "username": "string",
    "name": "string",
    "phone_number": "string",
    "room_number": "string",
    "score": number,
    "group": "string",
    "group_code": "string",
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
]
```

#### 5. GetUserByUsernameHandler
**Endpoint:** `/api/users/by-username`  
**Method:** GET  
**Authentication:** Required (JWT Token)  
**Query Parameters:**  
- `username`: User's username  

**Models Used:**
- User

**Response:**
```json
{
  "id": "string",
  "username": "string",
  "name": "string",
  "phone_number": "string",
  "room_number": "string",
  "score": number,
  "group": "string",
  "group_code": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

#### 6. GetUsersByScoreHandler
**Endpoint:** `/api/users/by-score`  
**Method:** GET  
**Authentication:** Required (JWT Token)  
**Models Used:**
- User

**Response:**
```json
[
  {
    "id": "string",
    "username": "string",
    "name": "string",
    "phone_number": "string",
    "room_number": "string",
    "score": number,
    "group": "string",
    "group_code": "string",
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
]
```

### Group Endpoints

#### 7. CreateGroupHandler
**Endpoint:** `/api/groups`  
**Method:** POST  
**Authentication:** Required (JWT Token)  
**Request Body:**
```json
{
  "name": "string"
}
```
**Models Used:**
- Group

**Response:**
```json
{
  "id": "string",
  "name": "string",
  "group_code": "string",
  "members": ["string"],
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

#### 8. JoinGroupHandler
**Endpoint:** `/api/groups/join`  
**Method:** POST  
**Authentication:** Required (JWT Token)  
**Request Body:**
```json
{
  "username": "string",
  "group_name": "string (optional)",
  "groupCode": "string (optional)",
  "roomNo": "string (optional)"
}
```
**Models Used:**
- User
- Group

**Response:**
```json
{
  "success": true,
  "message": "Successfully joined group"
}
```

#### 9. GetGroupMembersHandler
**Endpoint:** `/api/groups/members`  
**Method:** GET  
**Authentication:** Required (JWT Token)  
**Query Parameters:**  
- `group_name`: Group name (optional)
- `group_code`: Group code (optional)

**Models Used:**
- Group
- User

**Response:**
```json
[
  {
    "id": "string",
    "username": "string",
    "name": "string",
    "phone_number": "string",
    "room_number": "string",
    "score": number
  }
]
```

### Chore Endpoints

#### 10. CreateIndividualChoreHandler
**Endpoint:** `/api/chores/individual`  
**Method:** POST  
**Authentication:** Required (JWT Token)  
**Request Body:**
```json
{
  "title": "string",
  "description": "string",
  "group_name": "string",
  "assigned_to": "string",
  "due_date": "timestamp",
  "points": number
}
```
**Models Used:**
- Chore
- User
- Group

**Response:**
```json
{
  "id": "string",
  "title": "string",
  "description": "string",
  "group_id": "string",
  "assigned_to": "string",
  "due_date": "timestamp",
  "points": number,
  "status": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

#### 11. CreateRecurringChoreHandler
**Endpoint:** `/api/chores/recurring`  
**Method:** POST  
**Authentication:** Required (JWT Token)  
**Request Body:**
```json
{
  "title": "string",
  "description": "string",
  "group_name": "string",
  "frequency": "string (daily/weekly/biweekly/monthly)",
  "points": number
}
```
**Models Used:**
- RecurringChore
- Group

**Response:**
```json
{
  "id": "string",
  "title": "string",
  "description": "string",
  "group_id": "string",
  "frequency": "string",
  "points": number,
  "is_active": boolean,
  "next_assignment": "timestamp",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

#### 12. GetUserChoresHandler
**Endpoint:** `/api/chores/user`  
**Method:** GET  
**Authentication:** Required (JWT Token)  
**Query Parameters:**  
- `username`: User's username  

**Models Used:**
- User
- Chore

**Response:**
```json
[
  {
    "id": "string",
    "title": "string",
    "description": "string",
    "group_id": "string",
    "assigned_to": "string",
    "due_date": "timestamp",
    "points": number,
    "status": "string",
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
]
```

#### 13. GetGroupChoresHandler
**Endpoint:** `/api/chores/group`  
**Method:** GET  
**Authentication:** Required (JWT Token)  
**Query Parameters:**  
- `group_name`: Group name

**Models Used:**
- Group
- Chore

**Response:**
```json
[
  {
    "id": "string",
    "title": "string",
    "description": "string",
    "group_id": "string",
    "assigned_to": "string",
    "due_date": "timestamp",
    "points": number,
    "status": "string",
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
]
```

#### 14. GetGroupRecurringChoresHandler
**Endpoint:** `/api/chores/group/recurring`  
**Method:** GET  
**Authentication:** Required (JWT Token)  
**Query Parameters:**  
- `group_name`: Group name  

**Models Used:**
- Group
- RecurringChore

**Response:**
```json
[
  {
    "id": "string",
    "title": "string",
    "description": "string",
    "group_id": "string",
    "frequency": "string",
    "points": number,
    "is_active": boolean,
    "next_assignment": "timestamp",
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
]
```

#### 15. CompleteChoreHandler
**Endpoint:** `/api/chores/complete`  
**Method:** POST  
**Authentication:** Required (JWT Token)  
**Request Body:**
```json
{
  "chore_id": "string",
  "username": "string"
}
```
**Models Used:**
- Chore
- User
- ChoreCompletion

**Response:**
```json
{
  "points_earned": number,
  "new_score": number
}
```

#### 16. UpdateChoreHandler
**Endpoint:** `/api/chores/update`  
**Method:** PUT  
**Authentication:** Required (JWT Token)  
**Request Body:**
```json
{
  "chore_id": "string",
  "title": "string (optional)",
  "description": "string (optional)",
  "assigned_to": "string (optional)",
  "due_date": "timestamp (optional)",
  "points": number (optional)
}
```
**Models Used:**
- Chore
- User

**Response:**
```json
{
  "id": "string",
  "title": "string",
  "description": "string",
  "group_id": "string",
  "assigned_to": "string",
  "due_date": "timestamp",
  "points": number,
  "status": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

#### 17. DeleteChoreHandler
**Endpoint:** `/api/chores/delete`  
**Method:** DELETE  
**Authentication:** Required (JWT Token)  
**Query Parameters:**  
- `chore_id`: ID of the chore to delete

**Models Used:**
- Chore

**Response:**
```json
{
  "success": true,
  "message": "Chore deleted successfully"
}
```

#### 18. UpdateRecurringChoreHandler
**Endpoint:** `/api/chores/recurring/update`  
**Method:** PUT  
**Authentication:** Required (JWT Token)  
**Request Body:**
```json
{
  "recurring_chore_id": "string",
  "title": "string (optional)",
  "description": "string (optional)",
  "frequency": "string (optional)",
  "points": number (optional),
  "is_active": boolean (optional)
}
```
**Models Used:**
- RecurringChore

**Response:**
```json
{
  "id": "string",
  "title": "string",
  "description": "string",
  "group_id": "string",
  "frequency": "string",
  "points": number,
  "is_active": boolean,
  "next_assignment": "timestamp",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

#### 19. DeleteRecurringChoreHandler
**Endpoint:** `/api/chores/recurring/delete`  
**Method:** DELETE  
**Authentication:** Required (JWT Token)  
**Query Parameters:**  
- `recurring_chore_id`: ID of the recurring chore to delete  

**Models Used:**
- RecurringChore

**Response:**
```json
{
  "success": true,
  "message": "Recurring chore deleted successfully"
}
```

### Pantry Endpoints

#### 20. AddPantryItemHandler
**Endpoint:** `/api/pantry/add`  
**Method:** POST  
**Authentication:** Required (JWT Token)  
**Request Body:**
```json
{
  "name": "string",
  "quantity": number,
  "unit": "string",
  "category": "string",
  "expiration_date": "timestamp (optional)",
  "group_name": "string"
}
```
**Models Used:**
- PantryItem
- Group
- PantryHistory

**Response:**
```json
{
  "id": "string",
  "name": "string",
  "quantity": number,
  "unit": "string",
  "category": "string",
  "expiration_date": "timestamp",
  "group_id": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

#### 21. UsePantryItemHandler
**Endpoint:** `/api/pantry/use`  
**Method:** POST  
**Authentication:** Required (JWT Token)  
**Request Body:**
```json
{
  "item_id": "string",
  "quantity": number
}
```
**Models Used:**
- PantryItem
- PantryHistory
- PantryNotification

**Response:**
```json
{
  "remaining_quantity": number,
  "unit": "string"
}
```

#### 22. GetPantryItemsHandler
**Endpoint:** `/api/pantry/list`  
**Method:** GET  
**Authentication:** Required (JWT Token)  
**Query Parameters:**  
- `group_name`: Group name  
- `category`: Category filter (optional)  

**Models Used:**
- Group
- PantryItem

**Response:**
```json
[
  {
    "id": "string",
    "name": "string",
    "quantity": number,
    "unit": "string",
    "category": "string",
    "expiration_date": "timestamp",
    "expiration_status": "string", // "ok", "expiring_soon", "expired"
    "group_id": "string",
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
]
```

#### 23. DeletePantryItemHandler
**Endpoint:** `/api/pantry/remove/{item_id}`  
**Method:** DELETE  
**Authentication:** Required (JWT Token)  
**URL Parameters:**  
- `item_id`: ID of the pantry item to delete  

**Models Used:**
- PantryItem
- PantryHistory

**Response:**
```json
{
  "success": true,
  "message": "Pantry item removed successfully"
}
```

#### 24. GetPantryWarningsHandler
**Endpoint:** `/api/pantry/warnings`  
**Method:** GET  
**Authentication:** Required (JWT Token)  
**Query Parameters:**  
- `group_name`: Group name (optional)  
- `group_code`: Group code (optional)  

**Models Used:**
- Group
- PantryItem
- PantryNotification

**Response:**
```json
[
  {
    "item_id": "string",
    "name": "string",
    "current_quantity": number,
    "unit": "string",
    "warning_type": "string", // "low_stock" or "out_of_stock"
    "timestamp": "timestamp"
  }
]
```

#### 25. GetPantryExpiringHandler
**Endpoint:** `/api/pantry/expiring`  
**Method:** GET  
**Authentication:** Required (JWT Token)  
**Query Parameters:**  
- `group_name`: Group name (optional)  
- `group_code`: Group code (optional)  

**Models Used:**
- Group
- PantryItem
- PantryNotification

**Response:**
```json
[
  {
    "item_id": "string",
    "name": "string",
    "current_quantity": number,
    "unit": "string",
    "expiration_date": "timestamp",
    "days_remaining": number,
    "status": "string" // "expiring_soon" or "expired"
  }
]
```

#### 26. GetPantryShoppingListHandler
**Endpoint:** `/api/pantry/shopping-list`  
**Method:** GET  
**Authentication:** Required (JWT Token)  
**Query Parameters:**  
- `group_name`: Group name (optional)  
- `group_code`: Group code (optional)  

**Models Used:**
- Group
- PantryItem
- PantryNotification

**Response:**
```json
[
  {
    "name": "string",
    "unit": "string",
    "reason": "string" // "low_stock", "out_of_stock", or "expired"
  }
]
```

#### 27. GetPantryHistoryHandler
**Endpoint:** `/api/pantry/history`  
**Method:** GET  
**Authentication:** Required (JWT Token)  
**Query Parameters:**  
- `group_name`: Group name (optional)  
- `group_code`: Group code (optional)  
- `item_id`: Filter history by specific item (optional)  

**Models Used:**
- Group
- PantryHistory

**Response:**
```json
[
  {
    "id": "string",
    "item_name": "string",
    "action": "string", // "add", "use", "remove"
    "quantity": number,
    "unit": "string",
    "user_id": "string",
    "username": "string",
    "timestamp": "timestamp"
  }
]
```

#### 28. MarkNotificationReadHandler
**Endpoint:** `/api/pantry/notify/read`  
**Method:** POST  
**Authentication:** Required (JWT Token)  
**Request Body:**
```json
{
  "notification_id": "string"
}
```
**Models Used:**
- PantryNotification

**Response:**
```json
{
  "success": true,
  "message": "Notification marked as read"
}
```

#### 29. DeleteNotificationHandler
**Endpoint:** `/api/pantry/notify/delete`  
**Method:** DELETE  
**Authentication:** Required (JWT Token)  
**Query Parameters:**  
- `notification_id`: ID of the notification to delete  

**Models Used:**
- PantryNotification

**Response:**
```json
{
  "success": true,
  "message": "Notification deleted successfully"
}
```

### Shopping Cart Endpoints

#### 30. AddShoppingCartItemHandler
**Endpoint:** `/api/shopping-cart/add`  
**Method:** POST  
**Authentication:** Required (JWT Token)  
**Request Body:**
```json
{
  "item_name": "string",
  "quantity": number
}
```
**Models Used:**
- ShoppingCartItem
- User
- Group
- ShoppingCartActivity

**Response:**
```json
{
  "id": "string",
  "item_name": "string",
  "quantity": number,
  "user_id": "string",
  "username": "string",
  "group_id": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

#### 31. UpdateShoppingCartItemHandler
**Endpoint:** `/api/shopping-cart/update`  
**Method:** PUT  
**Authentication:** Required (JWT Token)  
**Request Body:**
```json
{
  "item_id": "string",
  "item_name": "string (optional)",
  "quantity": number (optional)
}
```
**Models Used:**
- ShoppingCartItem
- ShoppingCartActivity

**Response:**
```json
{
  "id": "string",
  "item_name": "string",
  "quantity": number,
  "user_id": "string",
  "username": "string",
  "group_id": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

#### 32. DeleteShoppingCartItemHandler
**Endpoint:** `/api/shopping-cart/delete/:item_id`  
**Method:** DELETE  
**Authentication:** Required (JWT Token)  
**URL Parameters:**  
- `item_id`: ID of the shopping cart item to delete  

**Models Used:**
- ShoppingCartItem
- ShoppingCartActivity

**Response:**
```json
{
  "success": true,
  "message": "Item removed from shopping cart"
}
```

#### 33. ListShoppingCartItemsHandler
**Endpoint:** `/api/shopping-cart/list`  
**Method:** GET  
**Authentication:** Required (JWT Token)  
**Query Parameters:**  
- `user_id`: Filter by specific user (optional)  

**Models Used:**
- User
- Group
- ShoppingCartItem

**Response:**
```json
[
  {
    "id": "string",
    "item_name": "string",
    "quantity": number,
    "user_id": "string",
    "username": "string",
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
]
```

#### 34. GetShoppingCartActivityHandler
**Endpoint:** `/api/shopping-cart/activity`  
**Method:** GET  
**Authentication:** Required (JWT Token)  
**Query Parameters:**  
- `group_name`: Group name (optional)  
- `group_code`: Group code (optional)  

**Models Used:**
- Group
- ShoppingCartActivity

**Response:**
```json
[
  {
    "id": "string",
    "action": "string", // "add", "update", "remove"
    "item_name": "string",
    "quantity": number,
    "previous_quantity": number,
    "user_id": "string",
    "username": "string",
    "is_read": boolean,
    "timestamp": "timestamp"
  }
]
```

#### 35. MarkActivityReadHandler
**Endpoint:** `/api/shopping-cart/activity/read`  
**Method:** POST  
**Authentication:** Required (JWT Token)  
**Request Body:**
```json
{
  "activity_id": "string"
}
```
**Models Used:**
- ShoppingCartActivity

**Response:**
```json
{
  "success": true,
  "message": "Activity marked as read"
}
``` 

## Middleware Documentation

The Cribb Backend application uses several middleware components to handle authentication, request validation, and access control. This section documents these middleware and how they are used throughout the application.

### Authentication Middleware

#### AuthMiddleware
**File:** `middleware/auth.go`  
**Purpose:** Verifies JWT tokens and authenticates users

**Functionality:**
- Extracts the JWT token from the `Authorization` header
- Validates the token signature using the application's secret key
- Extracts user claims (ID and username) from the token
- Stores user information in the request context for downstream handlers
- Returns 401 Unauthorized if any validation step fails

**Usage Example:**
```go
http.HandleFunc("/api/protected-route", middleware.AuthMiddleware(handlerFunction))
```

#### UserClaims
**Structure:**
```go
type UserClaims struct {
  ID       string
  Username string
}
```

**Helper Functions:**
- `GetUserFromContext(ctx context.Context) (UserClaims, bool)`: Retrieves user claims from request context

### CORS Middleware

#### CORSMiddleware
**File:** `middleware/auth.go`  
**Purpose:** Handles Cross-Origin Resource Sharing (CORS) for browser clients

**Functionality:**
- Sets appropriate CORS headers to allow requests from the frontend application
- Handles preflight requests (OPTIONS method)
- Supports credentials for authenticated requests
- Configured to allow requests from `http://localhost:4200`

**Usage Example:**
```go
http.HandleFunc("/api/endpoint", middleware.CORSMiddleware(handlerFunction))
```

### Access Control Middleware

#### GroupAccessControlMiddleware
**File:** `middleware/access_control.go`  
**Purpose:** Ensures users can only access resources from their own group

**Functionality:**
- Verifies the user belongs to the group specified in the request
- Works with both group name and group code parameters
- Stores verified group ID in the request context
- Returns 403 Forbidden if the user is not a member of the specified group

**Usage Example:**
```go
http.HandleFunc("/api/group-resources", middleware.AuthMiddleware(
  middleware.GroupAccessControlMiddleware(handlerFunction)))
```

**Helper Functions:**
- `GetVerifiedGroupID(ctx context.Context) (primitive.ObjectID, bool)`: Retrieves verified group ID from context

#### ResourceOwnershipMiddleware
**File:** `middleware/access_control.go`  
**Purpose:** Ensures users can only modify resources they own

**Functionality:**
- Extracts resource ID from URL path or query parameters
- Verifies the user is the owner of the resource
- Bypasses ownership check for GET requests (read operations)
- Returns 403 Forbidden if the user does not own the resource

**Parameters:**
- `resourceCollection`: MongoDB collection name where the resource is stored
- `resourceIDParam`: Name of the parameter containing the resource ID

**Usage Example:**
```go
http.HandleFunc("/api/user-resources", middleware.AuthMiddleware(
  middleware.ResourceOwnershipMiddleware(handlerFunction, "resources", "resource_id")))
```

### Validation Middleware

#### ValidateRequest
**File:** `middleware/validation.go`  
**Purpose:** Validates request bodies against defined rules

**Functionality:**
- Validates JSON request bodies against struct validation tags
- Supports required field validation
- Supports minimum value validation for strings and numbers
- Returns detailed validation errors with field names and messages
- Preserves request body for downstream handlers

**Usage Example:**
```go
type CreateItemRequest struct {
  Name     string  `json:"name" validate:"required"`
  Quantity float64 `json:"quantity" validate:"required,min=0.1"`
}

http.HandleFunc("/api/items", middleware.AuthMiddleware(
  middleware.ValidateRequest(handlerFunction, CreateItemRequest{})))
```

**Validation Tags:**
- `required`: Field must not be empty
- `min=X`: Field must have a minimum length (string) or value (number)

### Middleware Chaining

Middleware in the Cribb Backend is designed to be chainable. A typical protected endpoint might use multiple middleware components:

```go
http.HandleFunc("/api/protected-resource",
  middleware.CORSMiddleware(
    middleware.AuthMiddleware(
      middleware.GroupAccessControlMiddleware(
        middleware.ValidateRequest(handlerFunction, RequestStruct{})))))
```

This chain:
1. Adds CORS headers for browser access
2. Authenticates the user via JWT
3. Verifies the user belongs to the specified group
4. Validates the request body
5. Passes control to the handler function if all checks pass 