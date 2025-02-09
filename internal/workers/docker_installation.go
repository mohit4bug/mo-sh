package workers

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/mohit4bug/mo-sh/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/crypto/ssh"
)

func ListenForDockerInstallationEvents(ch *amqp.Channel, db *sql.DB, workerCount int) {
	_, err := ch.QueueDeclare(
		config.DockerInstallationQueue, true, false, false, false, nil,
	)
	if err != nil {
		log.Println(err)
		return
	}

	for i := 0; i < workerCount; i++ {
		go func() {
			msgs, err := ch.Consume(config.DockerInstallationQueue, "", false, false, false, false, nil)
			if err != nil {
				log.Println(err)
				return
			}

			for msg := range msgs {
				serverID := string(msg.Body)

				tx, err := db.Begin()
				if err != nil {
					log.Println(err)
					continue
				}

				var (
					installationID string
					hostname       string
					privateKey     string
					port           int
				)

				err = tx.QueryRow(
					`WITH updated AS (
						UPDATE docker_installations 
						SET status = 'in_progress' 
						WHERE server_id = $1 AND status = 'not_started' 
						RETURNING id, server_id
					)
					SELECT u.id, s.hostname, s.port, k.key
					FROM updated u
					INNER JOIN servers s ON s.id = u.server_id
					INNER JOIN keys k ON k.id = s.key_id`,
					serverID,
				).Scan(&installationID, &hostname, &port, &privateKey)
				if err != nil {
					log.Println(err)
					tx.Rollback()
					continue
				}

				if err := deployDockerOnHost(serverID, hostname, privateKey, port); err != nil {
					_, err = tx.Exec(
						`UPDATE docker_installations
						 SET status = 'failed'
						 WHERE id = $1`,
						installationID,
					)
					if err != nil {
						log.Println(err)
						tx.Rollback()
						continue
					}

					if err := tx.Commit(); err != nil {
						log.Println(err)
						continue
					}
					continue
				}

				_, err = tx.Exec(
					`UPDATE docker_installations 
					 SET status = 'completed' 
					 WHERE id = $1`,
					installationID,
				)
				if err != nil {
					log.Println(err)
					tx.Rollback()
					continue
				}

				if err := tx.Commit(); err != nil {
					log.Println(err)
					continue
				}

				msg.Ack(false)
			}
		}()
	}
}

func deployDockerOnHost(serverID, hostname, privateKey string, port int) error {
	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", hostname, port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		return err
	}

	cmd := `
		for pkg in docker.io docker-doc docker-compose docker-compose-v2 podman-docker containerd runc; do sudo apt-get remove $pkg; done

		# Add Docker's official GPG key:
		sudo apt-get update
		sudo apt-get install ca-certificates curl
		sudo install -m 0755 -d /etc/apt/keyrings
		sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
		sudo chmod a+r /etc/apt/keyrings/docker.asc

		# Add the repository to Apt sources:
		echo \
		  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
		  $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}") stable" | \
		  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
		sudo apt-get update

		sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
	`

	// Start the command
	if err := session.Start(cmd); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Wait for both stdout and stderr to finish.
	handleServerLogs(serverID, &wg, stdoutPipe)
	handleServerLogs(serverID, &wg, stderrPipe)

	if err := session.Wait(); err != nil {
		return err
	}

	wg.Wait()

	return nil
}

func handleServerLogs(serverID string, wg *sync.WaitGroup, pipe io.Reader) {
	defer wg.Done()
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		log.Printf("%s: %s", serverID, scanner.Text())
		// TODO: Save logs to the database
	}
}
