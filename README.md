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

http  'https://www.clasohlson.com/no/cocheckout/getCartDataOnReload?variantProductCode=445689000' Cookie:COStoreCookie=200 


Feedback:
- send link to list doesn't work as expected
- auto-generate list every Sunday, with title 'Week X' where X is the week number.

# Editing fields
Looks like we can use [contenteditable](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Global_attributes/contenteditable), so we'll do `<h3 contenteditable="true">Title</h3>`. Not how this affects accessibility.


# What I've learned
- sqlite doesn't enforce foreign key constraints by defualt. Need `PRAGMA foreign_keys = ON;` to enable that.
- Meta viewport: https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/meta/name/viewport. Lies under `<html`, not under `<head>`. 
- css: padding 1 2 3 4 means top right bottom left.
- css: padding 1 2 means top/bottom and left/right.
- css: border-radius 1 2 3 4 means top-left, top-right, bottom-right, bottom-left.
