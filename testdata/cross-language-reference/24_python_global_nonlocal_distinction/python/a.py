def handler():
    pass

def outer():
    handler = lambda: None
    def inner():
        nonlocal handler
        handler()
    inner()

def use_global():
    global handler
    handler()
