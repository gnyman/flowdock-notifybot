>“Simplicity is the ultimate sophistication.” 
> >― Leonardo da Vinci


httpstream is an extremely simple library for consuming streaming HTTP API's. It was inspired by and originally forked from [https://github.com/araddon/httpstream](https://github.com/araddon/httpstream) which, in turn, was originally forked from [https://github.com/hoisie/twitterstream](https://github.com/hoisie/twitterstream)

While the interface is similar, this library has a 93% smaller (LOC) footprint and should be much easier to read through, understand and use.

### Usage

    package main

    import (
        "log"
        "github.com/njern/httpstream"
    )
    
    func main() {
        // Incoming data channel
	    stream := make(chan []byte, 1024)
	    // Error channel
	    done := make(chan error)
	    // Set up the streaming client.
	    client := httpstream.NewClient(func(line []byte) {
		    stream <- line
	    })

		err := client.Connect("http://www.some-streaming-endpoint.com", done)
		if err != nil {
			panic(err)
		}

		for {
			select {
			case event := <-stream:
				log.Println(string(event))
			case err := <-done:
				if err == nil {
				    // Connection was closed by the user
				    log.Println("Connection was closed")
				} else {
					// The connection died/closed because of an error
					panic(err)
				}
			}
		}
    }