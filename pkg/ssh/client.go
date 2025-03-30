package ssh

import (
	"bufio"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	Host    string
	Port    int
	User    string
	Key     []byte
	Timeout time.Duration
	conn    *ssh.Client
}

func NewClient(host string, port int, user string, key []byte) *Client {
	return &Client{
		Host:    host,
		Port:    port,
		User:    user,
		Key:     key,
		Timeout: 5 * time.Second,
	}
}

func (c *Client) Connect() error {
	signer, err := ssh.ParsePrivateKey(c.Key)
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User: c.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         c.Timeout,
	}

	address := fmt.Sprintf("%s:%d", c.Host, c.Port)
	conn, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) RunCommand(cmd string) (string, string, error) {
	if c.conn == nil {
		return "", "", fmt.Errorf("not connected")
	}

	session, err := c.conn.NewSession()
	if err != nil {
		return "", "", err
	}
	defer session.Close()

	var stdoutBuf, stderrBuf io.Reader

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return "", "", err
	}
	stdoutBuf = stdoutPipe

	stderrPipe, err := session.StderrPipe()
	if err != nil {
		return "", "", err
	}
	stderrBuf = stderrPipe

	if err := session.Start(cmd); err != nil {
		return "", "", err
	}

	stdoutCh := make(chan string)
	stderrCh := make(chan string)

	go func() {
		var output string
		scanner := bufio.NewScanner(stdoutBuf)
		for scanner.Scan() {
			output += scanner.Text() + "\n"
		}
		stdoutCh <- output
	}()

	go func() {
		var output string
		scanner := bufio.NewScanner(stderrBuf)
		for scanner.Scan() {
			output += scanner.Text() + "\n"
		}
		stderrCh <- output
	}()

	stdout := <-stdoutCh
	stderr := <-stderrCh

	if err := session.Wait(); err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

func (c *Client) ExecuteWithStreams(cmd string, stdoutCallback, stderrCallback func(string)) error {
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	session, err := c.conn.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	if err := session.Start(cmd); err != nil {
		return err
	}

	go streamOutput(stdout, stdoutCallback)
	go streamOutput(stderr, stderrCallback)

	return session.Wait()
}

func (c *Client) CheckCommand(cmd string) bool {
	if c.conn == nil {
		return false
	}

	session, err := c.conn.NewSession()
	if err != nil {
		return false
	}
	defer session.Close()

	return session.Run(cmd) == nil
}

func streamOutput(reader io.Reader, callback func(string)) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		callback(scanner.Text())
	}
}
