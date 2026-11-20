def helper():
    pass

def outer():
    helper = lambda: None
    def inner():
        helper = lambda: None
        helper()
    helper()
