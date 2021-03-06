package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	triton "github.com/joyent/triton-go"
	"github.com/joyent/triton-go/authentication"
	"github.com/joyent/triton-go/storage"
)

func main() {
	keyID := os.Getenv("SDC_KEY_ID")
	accountName := os.Getenv("SDC_ACCOUNT")
	keyPath := os.Getenv("SDC_KEY_FILE")

	privateKey, err := ioutil.ReadFile(keyPath)
	if err != nil {
		log.Fatalf("Couldn't find key file matching %s\n%s", keyID, err)
	}

	sshKeySigner, err := authentication.NewPrivateKeySigner(keyID, privateKey, accountName)
	if err != nil {
		log.Fatal(err)
	}

	config := &triton.ClientConfig{
		MantaURL:    os.Getenv("MANTA_URL"),
		AccountName: accountName,
		Signers:     []authentication.Signer{sshKeySigner},
	}
	client, err := storage.NewClient(config)
	if err != nil {
		log.Fatalf("NewClient: %s", err)
	}

	job, err := client.Jobs().Create(context.Background(), &storage.CreateJobInput{
		Name: "WordCount",
		Phases: []*storage.JobPhase{
			{
				Type: "map",
				Exec: "wc",
			},
			{
				Type: "reduce",
				Exec: "awk '{ l += $1; w += $2; c += $3 } END { print l, w, c }'",
			},
		},
	})
	if err != nil {
		log.Fatalf("CreateJob: %s", err)
	}

	fmt.Printf("Job ID: %s\n", job.JobID)

	err = client.Jobs().AddInputs(context.Background(), &storage.AddJobInputsInput{
		JobID: job.JobID,
		ObjectPaths: []string{
			fmt.Sprintf("/%s/stor/books/treasure_island.txt", accountName),
			fmt.Sprintf("/%s/stor/books/moby_dick.txt", accountName),
			fmt.Sprintf("/%s/stor/books/huck_finn.txt", accountName),
			fmt.Sprintf("/%s/stor/books/dracula.txt", accountName),
		},
	})
	if err != nil {
		log.Fatalf("AddJobInputs: %s", err)
	}

	err = client.Jobs().AddInputs(context.Background(), &storage.AddJobInputsInput{
		JobID: job.JobID,
		ObjectPaths: []string{
			fmt.Sprintf("/%s/stor/books/sherlock_holmes.txt", accountName),
		},
	})
	if err != nil {
		log.Fatalf("AddJobInputs: %s", err)
	}

	gjo, err := client.Jobs().Get(context.Background(), &storage.GetJobInput{
		JobID: job.JobID,
	})
	if err != nil {
		log.Fatalf("GetJob: %s", err)
	}

	fmt.Printf("%+v\n", gjo.Job)
	fmt.Printf("%+v\n", gjo.Job.Stats)

	err = client.Jobs().EndInput(context.Background(), &storage.EndJobInputInput{
		JobID: job.JobID,
	})
	if err != nil {
		log.Fatalf("EndJobInput: %s", err)
	}

	jobs, err := client.Jobs().List(context.Background(), &storage.ListJobsInput{})
	if err != nil {
		log.Fatalf("ListJobs: %s", err)
	}

	fmt.Printf("Number of jobs: %d\n", jobs.ResultSetSize)
	for _, j := range jobs.Jobs {
		fmt.Printf(" - %s\n", j.ID)
	}

	gjio, err := client.Jobs().GetInput(context.Background(), &storage.GetJobInputInput{
		JobID: job.JobID,
	})
	if err != nil {
		log.Fatalf("GetJobInput: %s", err)
	}
	defer gjio.Items.Close()

	fmt.Printf("Result set size: %d\n", gjio.ResultSetSize)
	outputsScanner := bufio.NewScanner(gjio.Items)
	for outputsScanner.Scan() {
		fmt.Printf(" - %s\n", outputsScanner.Text())
	}

	time.Sleep(10 * time.Second)

	gjoo, err := client.Jobs().GetOutput(context.Background(), &storage.GetJobOutputInput{
		JobID: job.JobID,
	})
	if err != nil {
		log.Fatalf("GetJobOutput: %s", err)
	}
	defer gjoo.Items.Close()

	fmt.Printf("Result set size: %d\n", gjoo.ResultSetSize)
	outputsScanner = bufio.NewScanner(gjoo.Items)
	for outputsScanner.Scan() {
		fmt.Printf(" - %s\n", outputsScanner.Text())
	}
}
