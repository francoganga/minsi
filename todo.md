TODO:


- func makeCrud(model) -> router/struct that contains something like map[string]http.Handler
    so then we could do things like get the handler_action from the request's query and then use object[handler_action] to respond.
