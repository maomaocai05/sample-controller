FROM docker.io/alpine:latest
  
RUN mkdir -p /app
COPY sample-controller /app
 
 
ENTRYPOINT ["/data/go/sample-controller"]
