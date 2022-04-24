package main

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/proto/tendermint/crypto"
)

// Structure for data passed to print response.
type response struct {
	// generic abci response
	Data []byte
	Code uint32
	Info string
	Log  string

	Query *queryResponse
}

type queryResponse struct {
	Key      []byte
	Value    []byte
	Height   int64
	ProofOps *crypto.ProofOps
}

func printResponse(cmd *cobra.Command, args []string, rsp response) {

	// Always print the status code.
	if rsp.Code == types.CodeTypeOK {
		fmt.Printf("-> code: OK\n")
	} else {
		fmt.Printf("-> code: %d\n", rsp.Code)

	}

	if len(rsp.Data) != 0 {
		// Do no print this line when using the commit command
		// because the string comes out as gibberish
		if cmd.Use != "commit" {
			fmt.Printf("-> data: %s\n", rsp.Data)
		}
		fmt.Printf("-> data.hex: 0x%X\n", rsp.Data)
	}
	if rsp.Log != "" {
		fmt.Printf("-> log: %s\n", rsp.Log)
	}

	if rsp.Query != nil {
		fmt.Printf("-> height: %d\n", rsp.Query.Height)
		if rsp.Query.Key != nil {
			fmt.Printf("-> key: %s\n", rsp.Query.Key)
			fmt.Printf("-> key.hex: %X\n", rsp.Query.Key)
		}
		if rsp.Query.Value != nil {
			fmt.Printf("-> value: %s\n", rsp.Query.Value)
			fmt.Printf("-> value.hex: %X\n", rsp.Query.Value)
		}
		if rsp.Query.ProofOps != nil {
			fmt.Printf("-> proof: %#v\n", rsp.Query.ProofOps)
		}
	}
}

// NOTE: s is interpreted as a string unless prefixed with 0x
func stringOrHexToBytes(s string) ([]byte, error) {
	if len(s) > 2 && strings.ToLower(s[:2]) == "0x" {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			err = fmt.Errorf("error decoding hex argument: %s\n", err.Error())
			return nil, err
		}
		return b, nil
	}

	if !strings.HasPrefix(s, "\"") || !strings.HasSuffix(s, "\"") {
		err := fmt.Errorf("invalid string arg: \"%s\". Must be quoted or a \"0x\"-prefixed hex string\n", s)
		return nil, err
	}

	return []byte(s[1 : len(s)-1]), nil
}
