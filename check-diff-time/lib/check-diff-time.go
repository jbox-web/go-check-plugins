package checkdifftime

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/mackerelio/checkers"
	"golang.org/x/crypto/ssh"
)

type sshOpts struct {
	Hostname     string  `short:"H" long:"hostname" default:"localhost" description:"Host name or IP Address"`
	Port         int     `short:"P" long:"port" default:"22" description:"Port number"`
	Timeout      float64 `short:"t" long:"timeout" default:"30" description:"Seconds before connection times out"`
	Warning      int64   `short:"w" long:"warning" default:"5" description:"Response time to result in warning status (seconds)"`
	Critical     int64   `short:"c" long:"critical" default:"10" description:"Response time to result in critical status (seconds)"`
	User         string  `short:"u" long:"user" description:"Login user name" env:"USER"`
	Password     string  `short:"p" long:"password" description:"Login password" env:"LOGIN_PASSWORD"`
	IdentityFile string  `short:"i" long:"identity" description:"Identity file (ssh private key)"`
	PassPhrase   string  `long:"passphrase" description:"Identity passphrase" env:"CHECK_SSH_IDENTITY_PASSPHRASE"`
}

// Do the plugin
func Do() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		os.Exit(1)
	}

	ckr := opts.run()
	ckr.Name = "Diff Time"
	ckr.Exit()
}

func parseArgs(args []string) (*sshOpts, error) {
	opts := &sshOpts{}
	_, err := flags.ParseArgs(opts, args)
	return opts, err
}

func decrypt(block *pem.Block, passphrase string) (*pem.Block, error) {
	data, err := x509.DecryptPEMBlock(block, []byte(passphrase))
	if err != nil {
		return nil, err
	}

	decryptedBlock := &pem.Block{
		Type:  block.Type,
		Bytes: data,
	}
	return decryptedBlock, nil
}

func readPrivateKey(file, passphrase string) ([]byte, error) {
	privateKey, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	block, rest := pem.Decode(privateKey)
	if len(rest) > 0 {
		return nil, errors.New("Invalid private key")
	}
	if !x509.IsEncryptedPEMBlock(block) {
		return privateKey, nil
	}

	block, err = decrypt(block, passphrase)
	if err != nil {
		return nil, err
	}

	privateKey = pem.EncodeToMemory(block)
	return privateKey, nil
}

func (opts *sshOpts) makeClientConfig() (*ssh.ClientConfig, error) {
	authenticities := make([]ssh.AuthMethod, 0, 1)
	if opts.Password != "" {
		authenticities = append(authenticities, ssh.Password(opts.Password))
	}
	if opts.IdentityFile != "" {
		data, err := readPrivateKey(opts.IdentityFile, opts.PassPhrase)
		if err != nil {
			return nil, err
		}

		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			return nil, err
		}

		authenticities = append(authenticities, ssh.PublicKeys(signer))
	}

	config := &ssh.ClientConfig{User: opts.User, Auth: authenticities, HostKeyCallback: ssh.InsecureIgnoreHostKey()}
	return config, nil
}

func (opts *sshOpts) dial(config *ssh.ClientConfig) (*ssh.Client, error) {
	addr := opts.Hostname + ":" + strconv.Itoa(opts.Port)
	timeout := opts.Timeout * float64(time.Second)
	conn, err := net.DialTimeout("tcp", addr, time.Duration(timeout))
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

func (opts *sshOpts) run() *checkers.Checker {
	// prevent changing output of some commands
	os.Setenv("LANG", "C")
	os.Setenv("LC_ALL", "C")

	config, err := opts.makeClientConfig()
	if err != nil {
		return checkers.Unknown(err.Error())
	}

	client, err := opts.dial(config)
	if err != nil {
		return checkers.Critical(err.Error())
	}

	session, err := client.NewSession()
	if err != nil {
		return checkers.Critical(err.Error())
	}

	output, err := session.Output("date +%s")
	if err != nil {
		return checkers.Unknown(err.Error())
	}

	outputStr := string(output)
	outputClean := strings.TrimRight(outputStr, "\r\n")

	remoteDate, err := strconv.ParseInt(outputClean, 10, 64)
	startDate := time.Now().Unix()

	diffTime := Abs(startDate - remoteDate)

	var checkState checkers.Status

	if diffTime > 0 && diffTime < opts.Warning {
		checkState = checkers.OK
	} else if diffTime >= opts.Warning && diffTime < opts.Critical {
		checkState = checkers.WARNING
	} else if diffTime >= opts.Critical {
		checkState = checkers.CRITICAL
	}

	message := fmt.Sprintf("Current date: %d (%s) - Remote date: %d (%s) - Diff time: %d", startDate, time.Unix(startDate, 0), remoteDate, time.Unix(remoteDate, 0), diffTime)
	return checkers.NewChecker(checkState, message)
}

// Abs returns the absolute value of x.
func Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
