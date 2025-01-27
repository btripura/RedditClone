package main

import (
    "flag"
    "fmt"
    "io"
    "log"
    "os"
    "github.com/asynkron/protoactor-go/actor"
    "github.com/asynkron/protoactor-go/remote"
    "reddit/engine"
    "reddit/rest"
)

func setupLogging() (*os.File, error) {
    // Create logs directory if it doesn't exist
    if err := os.MkdirAll("logs", 0755); err != nil {
        return nil, fmt.Errorf("failed to create logs directory: %v", err)
    }

    // Open log file
    f, err := os.OpenFile("logs/server.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
    if err != nil {
        return nil, fmt.Errorf("error opening log file: %v", err)
    }

    // Use MultiWriter to write logs to both file and console
    mw := io.MultiWriter(os.Stdout, f)
    log.SetOutput(mw)
    
    // Set log flags to include date, time with microseconds, and source file
    log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
    
    return f, nil
}

func main() {
    // Define command line flags
    httpPort := flag.Int("port", 8080, "REST API port")
    actorPort := flag.Int("actor-port", 8085, "Actor system port")
    flag.Parse()

    // Setup logging
    logFile, err := setupLogging()
    if err != nil {
        log.Fatalf("Failed to setup logging: %v", err)
    }
    defer logFile.Close()

    log.Printf("Starting Reddit Clone Server")
    log.Printf("HTTP Port: %d, Actor Port: %d", *httpPort, *actorPort)

    // Initialize actor system
    system := actor.NewActorSystem()
    log.Printf("Actor system initialized")
    
    // Configure remote actor system
    config := remote.Configure(
        "127.0.0.1",
        *actorPort,
        remote.WithEndpointWriterBatchSize(10),
        remote.WithEndpointWriterQueueSize(1000),
        remote.WithEndpointManagerBatchSize(10),
        remote.WithAdvertisedHost("127.0.0.1"),
    )

    // Start remote
    r := remote.NewRemote(system, config)
    r.Start()
    log.Printf("Remote actor system started")
    
    // Create and start social engine actor
    engine := engine.NewSocialEngine()
    props := actor.PropsFromProducer(func() actor.Actor {
        return engine
    })

    pid, err := system.Root.SpawnNamed(props, "social")
    if err != nil {
        log.Fatalf("Failed to start engine: %v", err)
    }
    log.Printf("Social engine actor spawned with PID: %v", pid)

    // Create and start REST API server
    server := rest.NewServer(pid, system)
    log.Printf("Starting REST server on port %d", *httpPort)
    
    // Start server and log any errors
    if err := server.Start(*httpPort); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}