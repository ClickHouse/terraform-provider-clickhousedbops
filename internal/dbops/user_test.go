package dbops

import (
	"context"
	"testing"

	"github.com/pingcap/errors"
)

const (
	testUserID   = "1e4a8a40-dd60-b0d9-7baa-d44f89302d8e"
	testUserName = "test_user"
)

const (
	getUserByIDSQL = "SELECT `name` FROM `system`.`users` WHERE (`id` = '1e4a8a40-dd60-b0d9-7baa-d44f89302d8e');"
	userInheritSQL = "SELECT `inherit_profile` FROM `system`.`settings_profile_elements` WHERE (`user_name` = 'test_user');"
	dropUserSQL    = "DROP USER `test_user`;"
)

func TestDeleteUser(t *testing.T) {
	tests := []struct {
		name    string
		script  []step
		wantErr bool
	}{
		{
			name: "drops the user when it exists",
			script: []step{
				{wantSQL: getUserByIDSQL, rows: map[string]string{"name": testUserName}},
				{wantSQL: userInheritSQL},
				{wantSQL: dropUserSQL},
			},
		},
		{
			name: "no-op when the user is genuinely absent",
			script: []step{
				{wantSQL: getUserByIDSQL},
			},
		},
		{
			name: "fails when the existence check errors instead of orphaning the user",
			// A failed read must never be mistaken for "user absent": reporting
			// success here would orphan the real user in ClickHouse.
			script: []step{
				{wantSQL: getUserByIDSQL, err: errors.New("error iterating over rows: connection reset by peer")},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, fake := newTestClient(t, tc.script)

			err := client.DeleteUser(context.Background(), testUserID, nil)

			if len(fake.steps) != 0 {
				t.Errorf("script not fully consumed, %d steps left", len(fake.steps))
			}
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
