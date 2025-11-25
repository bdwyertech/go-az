package az

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

type kubeExecCredential struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Spec       struct {
		Interactive bool `json:"interactive"`
	} `json:"spec"`
	Status struct {
		ExpirationTimestamp time.Time `json:"expirationTimestamp"`
		Token               string    `json:"token"`
	} `json:"status"`
}

func GetKubeCred(ctx context.Context, opts AccessTokenOptions) (token kubeExecCredential, err error) {
	const (
		apiV1      string = "client.authentication.k8s.io/v1"
		apiV1beta1 string = "client.authentication.k8s.io/v1beta1"
	)
	if env := os.Getenv("KUBERNETES_EXEC_INFO"); env != "" {
		if err = json.Unmarshal([]byte(env), &token); err != nil {
			return token, fmt.Errorf("cannot unmarshal %q to kubeExecCredential: %w", env, err)
		}
	}
	switch token.APIVersion {
	case "":
		token.APIVersion = apiV1beta1
	case apiV1, apiV1beta1:
		break
	default:
		return token, fmt.Errorf("api version: %s is not supported", token.APIVersion)
	}

	popts := policy.TokenRequestOptions{
		Scopes: opts.Scope,
	}
	if opts.Resource != "" {
		popts.Scopes = append(popts.Scopes, opts.Resource+"/.default")
	}

	t, err := GetToken(ctx, TokenOptions{
		TokenRequestOptions: popts,
		ClientID:            opts.Client,
		TenantID:            opts.Tenant,
		ForceInteractive:    false,
		PreferredUsername:   "",
	})
	if err != nil {
		return
	}

	token.Status.ExpirationTimestamp = t.ExpiresOn
	token.Status.Token = t.AccessToken

	return
}
