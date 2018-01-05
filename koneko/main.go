package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"syscall"

	"math/big"

	"github.com/labstack/gommon/log"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
)

func generateSecret() {
	const (
		alphaNumeric = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
		length       = 64
	)
	maxRand := int64(len(alphaNumeric))
	chars := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(maxRand))
		if err != nil {
			log.Fatal(err)
		}
		chars[i] = alphaNumeric[n.Int64()]
	}

	fmt.Println(string(chars))
}

func passwordDigest(cost int) {
	fmt.Printf("Digest your password(cost = %v)\n", cost)
	if cost < bcrypt.MinCost || bcrypt.MaxCost < cost {
		log.Fatal(bcrypt.InvalidCostError(cost))
	}
	fmt.Print("Password: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println()
	fmt.Println("Generating...")
	digest, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(digest))
}

func help() {
	fmt.Fprint(os.Stderr, `usage: koneko <command>
commands:
	secret		Generate secret key
	password	Generate password digest
	help		Print usage

See 'coneko [command] -h' to read about a specific subcommand.
`)
}

func main() {
	if len(os.Args) < 2 {
		help()
		return
	}

	passwordFs := flag.NewFlagSet("password", flag.ExitOnError)
	cost := passwordFs.Int("c", 14, "cost of bcrypt")
	secretFs := flag.NewFlagSet("secret", flag.ExitOnError)

	var args []string
	if 2 < len(os.Args) {
		args = os.Args[2:]
	}
	switch os.Args[1] {
	case "secret":
		secretFs.Parse(args)
		if secretFs.Parsed() {
			generateSecret()
		}
	case "password":
		passwordFs.Parse(args)
		if passwordFs.Parsed() {
			passwordDigest(*cost)
		}
	case "help":
		help()
	default:
		fmt.Fprintln(os.Stderr, "unknown command: ", os.Args[1])
		help()
	}
}
