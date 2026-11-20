def target():
    pass

def outer():
    def inner():
        target = lambda: None
        target()
    inner()
