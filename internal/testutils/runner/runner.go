package runner

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	testutils "github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/compose"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/dbopsclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/factories"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/providerconfig"
	internalstatecheck "github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/statecheck"
)

type TestCase struct {
	Name            string
	ChEnv           map[string]string
	Protocol        string
	ClusterName     *string
	Resource        string
	ResourceName    string
	ResourceAddress string

	CheckNotExistsFunc  func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]string) (bool, error)
	CheckAttributesFunc func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error
}

func RunTests(t *testing.T, tests []TestCase) {
	if os.Getenv("TF_ACC") != "1" {
		fmt.Println("Skipping test because TF_ACC is not set to 1")
		return
	}
	ctx := context.Background()

	// Prepare docker compose to run local clickhouse cluster.
	dcm := testutils.NewDockerComposeManager("../../../tests")

	for _, tc := range tests {
		// Run in an anonymous func to be able to use defer to take down clickhouse cluster
		// at the end of each test.
		func() {
			// Start CH cluster using docker compose.
			if err := dcm.Up(tc.ChEnv); err != nil {
				t.Fatal(err)
			}

			defer func() {
				// Take clickhouse cluster down.
				err := dcm.Down()
				if err != nil {
					t.Fatal(err)
				}
			}()

			dbopsClient, connSettings, err := dbopsclient.NewDbopsClient(tc.Protocol)
			if err != nil {
				t.Fatal(err)
			}

			providerCfg, err := providerconfig.ProviderConfig(tc.Protocol, connSettings.Host, connSettings.Port, connSettings.Username, connSettings.Password)
			if err != nil {
				t.Fatal(err)
			}

			t.Run(tc.Name, func(t *testing.T) {
				resource.Test(t, resource.TestCase{
					ProtoV6ProviderFactories: factories.ProviderFactories(),
					CheckDestroy: func(s *terraform.State) error {
						for address, r := range s.RootModule().Resources {
							if tc.ResourceAddress == address {
								exists, err := tc.CheckNotExistsFunc(ctx, dbopsClient, tc.ClusterName, r.Primary.Attributes)
								if err != nil {
									return err
								}

								if exists {
									return fmt.Errorf("expected resource to NOT exist, but it does")
								}

								return nil
							}
						}

						return fmt.Errorf("root module has no resource %q", tc.ResourceAddress)
					},
					Steps: []resource.TestStep{
						{
							// Combine the provider definition and the resourcePtr definition.
							Config: fmt.Sprintf("%s\n%s", providerCfg, tc.Resource),
							ConfigStateChecks: []statecheck.StateCheck{
								// Compare the state with the actual resource.
								internalstatecheck.NewGetAttributes(tc.ResourceAddress, func(attrs map[string]interface{}) error {
									return tc.CheckAttributesFunc(ctx, dbopsClient, tc.ClusterName, attrs)
								}),
							},
						},
					},
				})
			})
		}()
	}
}
