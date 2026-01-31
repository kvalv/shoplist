# Building another shopping list

I want this to be different:

- Simpler architecture for code
- Webhooks towards Home Assistant, so I can interract from there. For example, I want to
  have a button in Home Assistant that I can click, and when that happens, a notification
  is sent to my wife that I'll go shopping that day. She better fill out any remaining items
  soon.
- Clas Ohlson mode. If the cart is specific to Clas Ohlson, we should check item availability
  and get the item location in the store. I typically spend way too much time searching for
  items in that store.
- More event-driven architecture. The backend should drive application state, and we should
  be able to hand off tasks to the LLM to enrich data (in particular for Clas Ohlson).
- Use templating to render HTML pages, and use datastar, see how far I can get with that.


```
templ generate --watch --proxy="http://localhost:8080" --cmd="go run ."
```


- The initial load works fine, as I render the full page. Howeer, I'm struggling setting up
  an sse event handler that "re-renders" the page on updates.
