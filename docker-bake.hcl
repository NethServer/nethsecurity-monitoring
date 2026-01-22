target "base" {
    cache-from = [
        { type = "gha" }
    ]
    cache-to = [
        { type = "gha", mode = "max" }
    ]
}

target "test" {
    inherits = ["base"]
    target = "test"
    output = [
        { type = "cacheonly" }
    ]
}

target "build" {
    inherits = ["base"]
    target = "build"
    output = [
        { type = "cacheonly" }
    ]
}

target "dist" {
    inherits = ["base"]
    target = "dist"
    output = [
        { type = "local", dest = "." }
    ]
}
