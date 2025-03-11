package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

func main() {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"192.168.157.133:2379"},
	})
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	// do a connection check first, otherwise it will hang infinitely on newSession
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = cli.MemberList(ctx)
	if err != nil {
		log.Fatalf("failed to reach etcd: %s", err)
	}

	session, err := concurrency.NewSession(cli, concurrency.WithTTL(1))
	if err != nil {
		log.Fatalf("failed to create a session: %s", err)
	}
	go func() {
		time.Sleep(time.Second * 10)
		log.Println("Session closed")
		session.Close()
	}()

	log.Print("created etcd client and session")

	locker := concurrency.NewLocker(session, "/lock")

	fmt.Println("Enter commands")
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())

		switch command {
		case "lock", "l":
			locker.Lock()
			log.Println("Lock acquired")
		case "unlock", "u":
			locker.Unlock()
			log.Println("Lock released")
		case "expire", "e":
			session.Close()
			log.Println("Session closed")
		case "quit", "q":
			fmt.Println("Quit program")
			return
		default:
			fmt.Println("Unknown command.")
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading input: %v", err)
	}
}
