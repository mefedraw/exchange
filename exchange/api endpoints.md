👤 **UserHandler**

✅ **POST** `user/api/user/register`  
**Request:**
```json
{
  "email": "user@example.com",
  "password": "securePass123"
}
```
**Response – 201 Created:**
```json
{
  "id": 1
}
```
**Response – 409 Conflict:**
```json
{
  "error": "User already exists"
}
```
**Response – 400 | 500:**
```json
{
  "error": "Invalid email or password format"
}
```

✅ **POST** `user/api/user/login`  
**Request:**
```json
{
  "email": "user@example.com",
  "password": "securePass123"
}
```
**Response – 200 OK:**
```json
{
  "id": 1,
  "email": "user@example.com"
}
```
**Response – 401 Unauthorized:**
```json
{
  "error": "Invalid email or password"
}
```

✅ **GET** `user/api/user/balance`  
**Request:**
```json
{
  "user_id": 1
}
```
**Response – 200 OK:**
```json
{
  "user_id": 1,
  "balance": "1000.00"
}
```

✅ **POST** `user/api/user/balance/increase`  
**Request:**
```json
{
  "id": 1,
  "amount": "500.00"
}
```
**Response – 200 OK:**
```json
{
  "user_id": 1,
  "balance": "1500.00"
}
```

✅ **POST** `user/api/user/balance/decrease`  
**Request:**
```json
{
  "id": 1,
  "amount": "100.00"
}
```
**Response – 200 OK:**
```json
{
  "user_id": 1,
  "balance": "1400.00"
}
```

📈 **TradeHandler**

✅ **POST** `trade/api/trade/open`  
**Request:**
```json
{
  "user_id": 1,
  "ticker": "AAPL",
  "order_type": "buy",
  "margin": "100.00",
  "leverage": 5
}
```
**Response – 201 Created:**
```json
{
  "order_id": "uuid-value-here"
}
```
**Response – 400 | 500:**
```json
{
  "error": "Margin must be positive"
}
```

✅ **POST** `trade/api/trade/close`  
**Request:**
```json
{
  "order_id": "uuid-value-here",
  "ticker": "AAPL"
}
```
**Response – 200 OK:**
```json
{
  "order_id": "uuid-value-here"
}
```
**Response – 404 Not Found:**
```json
{
  "error": "Order not found"
}
```

✅ **GET** `trade/api/trade/orders`  
**Request:**
```json
{
  "id": 1
}
```
**Response – 200 OK:**
```json
{
  "orders": [
    {
      "order_id": "uuid",
      "user_id": 1,
      "ticker": "AAPL",
      "order_type": "buy",
      "status": "open",
      "created_at": "2024-12-06T12:34:56Z"
    }
  ]
}
```