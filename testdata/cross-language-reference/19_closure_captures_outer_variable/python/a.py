def target():
    pass

def outer():
    target = lambda: None
    def inner():
        target()
    inner()
