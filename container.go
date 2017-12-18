package main

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type Container struct {
	Stdin        chan string
	Stdout       chan string
	ID           string
	Client       *client.Client
	Running      bool
	InputStream  types.HijackedResponse
	OutputStream types.HijackedResponse
}

func (c *Container) Start(image string) {
	client, err := client.NewEnvClient()

	if err != nil {
		panic(err)
	}

	c.Client = client
	c.Stdin = make(chan string)
	c.Stdout = make(chan string)

	container, containerErr := client.ContainerCreate(
		context.Background(),
		&container.Config{
			Image:           image,
			AttachStdin:     true,
			AttachStdout:    true,
			OpenStdin:       true,
			Tty:             true,
			NetworkDisabled: true,
			// Cmd:          []string{"perl", "-e", "while(<STDIN>) {print $_;}"},
		},
		&container.HostConfig{},
		&network.NetworkingConfig{},
		"")

	if containerErr != nil {
		fmt.Print(containerErr)
		c.Stdout <- fmt.Sprint(containerErr)
		return
	}

	inputStream, err := c.Client.ContainerAttach(context.Background(), container.ID, types.ContainerAttachOptions{Stdin: true, Stream: true})
	if err != nil {
		panic(err)
	}
	outputStream, err := c.Client.ContainerAttach(context.Background(), container.ID, types.ContainerAttachOptions{Stdout: true, Stderr: true, Stream: true})

	if err != nil {
		panic(err)
	}
	err = client.ContainerStart(context.Background(), container.ID, types.ContainerStartOptions{})

	if err != nil {
		panic(err)
	}

	c.Running = true
	c.ID = container.ID

	c.InputStream = inputStream
	c.OutputStream = outputStream

	go c.ReadInput()
	go c.ReadOutput()
	go c.WaitForExit()
}

func (c *Container) ReadInput() {

	for msg := range c.Stdin {
		c.InputStream.Conn.Write([]byte(msg))
	}

}

func (c *Container) ReadOutput() {

	for {
		str, readErr := c.OutputStream.Reader.ReadByte()
		if readErr != nil {
			return
		}
		c.Stdout <- string(str)
	}

}

func (c *Container) Stop() {
	err := c.Client.ContainerStop(context.Background(), c.ID, nil)
	if err != nil {
		panic(err)
	}
}

func (c *Container) WaitForExit() {
	c.Client.ContainerWait(context.Background(), c.ID, container.WaitConditionNotRunning)
	c.Teardown()
}

func (c *Container) Teardown() {
	if c.Running {

		resChan, errChan := c.Client.ContainerWait(context.Background(), c.ID, container.WaitConditionNotRunning)

		select {
		case err := <-errChan:
			if err != nil {
				panic(err)
			}
		case <-resChan:
			fmt.Printf("%s exited\n", c.ID)
		}

		// output, _ := c.Client.ContainerLogs(context.Background(), c.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
		// io.Copy(os.Stdout, output)

		c.Running = false
		c.Client.ContainerRemove(context.Background(), c.ID, types.ContainerRemoveOptions{})
	}
}
