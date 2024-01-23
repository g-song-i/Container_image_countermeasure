package main

import (
	"context"
	"log"
	"os"
	"os/exec"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/container"
)

func verifyImageSignature(image string) bool {
	homeDir := os.Getenv("HOME")
	pubKeyPath := homeDir + "/cosign.pub"

	cmd := exec.Command("cosign", "verify", "--key", pubKeyPath, image)
	output, err := cmd.CombinedOutput()
	// log.Printf("cosign output: %s", string(output))

	if err != nil {
		log.Printf("cosign error: %v", err)
		return false
	}
	return true
}

func main() {
	cli, err := client.NewClientWithOpts(client.WithVersion("1.43"))
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	ctx := context.Background()
	msgs, errs := cli.Events(ctx, types.EventsOptions{})

	for {
		select {
		case err := <-errs:
			if err != nil {
				log.Fatal(err)
			}
		case msg := <-msgs:
			if msg.Type == "container" && msg.Action == "create" {
				var containerInfo types.ContainerJSON
				containerInfo, err = cli.ContainerInspect(ctx, msg.ID)
				if err != nil {
					log.Fatal(err)
				}

				image := containerInfo.Config.Image
				if !verifyImageSignature(image) {
					log.Printf("Image signature verification failed for %s. Stopping container.", image)
					stopOptions := container.StopOptions{
						Timeout: nil,
					}
					if err := cli.ContainerStop(ctx, msg.ID, stopOptions); err != nil {
						log.Printf("Failed to stop container: %s", err)
					}
				} else {
					log.Printf("Image signature verification succeeded for %s", image)
				}
			}
		}
	}
}