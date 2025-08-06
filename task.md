1. Update db/models.go , adding new models. Use https://github.com/shopspring/decimal library to handle all money-related fields. With 2 digits after comma.
  a. db.File should have many db.Receipts
  b. db.Receipt should have next fields
    - totalBeforeTax : decimal
    - tax : decimal
    - totalAfterTax : decimal
    - origin : text
    - recipient : text
    - details : text
    - summary : text
    - default fields from gorm
    - fileId : linked to db.File
  c. db.Receipt should have many db.Product
  d. db.Product should have next fields
    - totalBeforeTax : decimal
    - tax : decimal
    - totalAfterTax : decimal
    - title : text
    - details : text
    - summary : text
    - default fields from gorm
    - receiptID : linked to db.File
  e. db.Category
    - title : text
    - details : text
    - default fields from gorm
  f. db.ProductCategory
    - Make many-to-many relation between db.Product and db.Category
  g. db.Product has many db.Category and vice versa

2. Working with llm/chat.go
  a. On receiving db.Message from user:
    - create corresponding db.ExposedFile for each file.
    - for each file, provide a link to exposed file to openai and ask it to extract necessary data from the file provided
    - create corresponding models for each file with results of analysis
    - ask openai to prepare a short response with a short summary of files we received
    - respond it to the user
