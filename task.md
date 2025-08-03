DONE: # Rework how messenger clients are working. Located at messenger/ folder

DONE: 1. Messenger should have a method `.OnMessage(func(ctx context.Context, msg db.Message, response chan<- string))`.
DONE: 2. That function should be stored in messenger instance and called once a message is received.

DONE: # Rework messenger/telegram

DONE: 1. Instead of sending MessageRequest through the channel - call the function passed through `OnMessage`

DONE: # Following requirements should be satisfied:

DONE: 1. Responder should wait for a response in a non-blocking mode, showing animation of `Thinking...`

DONE: # Make changes in chatter/

DONE: 1. It should pass a function to `messenger` to process messages. On new message - it should call `llmc.HandleMessage(ctx, message, response)`

DONE: # Make changes to llm/

DONE: 1. Interface should have a method `llmc.HandleMessage(ctx context.Context, message db.Message, response chan<- string)`
DONE: 2. Rework llm/openai to handle the new flow
DONE: 3. Each user should have its own chat inside llmc to handle chatting efficiently.
DONE: 4. Every new message from user to llm or from llm to user should be saved as db.Message.
DONE: 5. If chat is missing - it should be created, loading up the history of messages for that particular user.
