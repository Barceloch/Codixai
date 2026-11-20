def helper():
    pass

def outer():
    def inner(helper):
        helper()
