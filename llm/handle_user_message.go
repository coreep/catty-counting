package llm

// .toai [
// configure a connection with gemini api. There is ENV variable GEMINI_API_KEY
// create a basic prompt which should be an instriuctions put at the beginning of each request, defining the purpose of the bot and what it should and shouldn't do
// create a function `HandleUserMessage`. It is supposed to answer to user's message. It should also receive a list of strings, which contains of previous messages from user and your answers, to improve your context knowledge. The response from llm should be "streamed" outside, so it could be displayed at the recepient
// ]
