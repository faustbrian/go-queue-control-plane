package postgres

import (
	"context"
	"strings"
	"testing"

	migrations "github.com/faustbrian/go-migrations/v4"
)

func TestMigrationSourceLoadsTenantScopedControlSchema(t *testing.T) {
	t.Parallel()

	source, err := MigrationSource()
	if err != nil {
		t.Fatalf("MigrationSource() error = %v", err)
	}

	migrations, err := source.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(migrations) != 9 {
		t.Fatalf("len(Load()) = %d, want 9", len(migrations))
	}

	migration := migrations[0]
	if migration.Version() != 1 || migration.Name() != "control_plane" {
		t.Fatalf("migration identity = %d/%q, want 1/control_plane", migration.Version(), migration.Name())
	}

	up := migration.UpSQL()
	for _, required := range []string{
		"CREATE TABLE queue_control_commands",
		"PRIMARY KEY (tenant_id, idempotency_key)",
		"CREATE TABLE queue_control_desired_states",
		"FOREIGN KEY (tenant_id, command_key)",
		"CREATE TABLE queue_control_audit_events",
		"CHECK (octet_length(previous_hash) = 32)",
		"CHECK (octet_length(hash) = 32)",
		"UNIQUE (tenant_id, hash)",
		"CREATE TABLE queue_control_audit_anchors",
		"CHECK (revision > 0)",
		"CREATE INDEX queue_control_commands_history_idx",
		"CREATE INDEX queue_control_audit_history_idx",
	} {
		if !strings.Contains(up, required) {
			t.Fatalf("UpSQL() missing %q", required)
		}
	}
	for _, forbidden := range []string{"redis", "valkey", "payload bytea"} {
		if strings.Contains(strings.ToLower(up), forbidden) {
			t.Fatalf("UpSQL() unexpectedly contains %q", forbidden)
		}
	}

	down := migration.DownSQL()
	for _, required := range []string{
		"DROP TABLE queue_control_audit_anchors",
		"DROP TABLE queue_control_audit_events",
		"DROP TABLE queue_control_desired_states",
		"DROP TABLE queue_control_commands",
	} {
		if !strings.Contains(down, required) {
			t.Fatalf("DownSQL() missing %q", required)
		}
	}

	statusMigration := migrations[1]
	if statusMigration.Version() != 2 || statusMigration.Name() != "command_outcomes" {
		t.Fatalf(
			"status migration identity = %d/%q, want 2/command_outcomes",
			statusMigration.Version(), statusMigration.Name(),
		)
	}
	for _, required := range []string{
		"DROP CONSTRAINT queue_control_commands_status_check",
		"DROP CONSTRAINT queue_control_commands_check1",
		"'unsupported'",
		"'timed_out'",
		"'partial'",
	} {
		if !strings.Contains(statusMigration.UpSQL(), required) {
			t.Fatalf("status UpSQL() missing %q", required)
		}
	}

	identifierMigration := migrations[2]
	if identifierMigration.Version() != 3 || identifierMigration.Name() != "command_identifiers" {
		t.Fatalf(
			"identifier migration = %d/%q, want 3/command_identifiers",
			identifierMigration.Version(), identifierMigration.Name(),
		)
	}
	for _, required := range []string{
		"ADD COLUMN command_id uuid",
		"SET command_id = gen_random_uuid()",
		"UNIQUE (tenant_id, command_id)",
		"FOREIGN KEY (tenant_id, command_id)",
	} {
		if !strings.Contains(identifierMigration.UpSQL(), required) {
			t.Fatalf("identifier UpSQL() missing %q", required)
		}
	}

	auditMigration := migrations[3]
	if auditMigration.Version() != 4 || auditMigration.Name() != "sensitive_access_audit" {
		t.Fatalf(
			"audit migration = %d/%q, want 4/sensitive_access_audit",
			auditMigration.Version(), auditMigration.Name(),
		)
	}
	for _, required := range []string{
		"ADD COLUMN command_id uuid",
		"ALTER COLUMN idempotency_key DROP NOT NULL",
		"DROP CONSTRAINT queue_control_audit_events_tenant_id_idempotency_key_fkey",
	} {
		if !strings.Contains(auditMigration.UpSQL(), required) {
			t.Fatalf("audit UpSQL() missing %q", required)
		}
	}

	lifecycleMigration := migrations[4]
	if lifecycleMigration.Version() != 5 || lifecycleMigration.Name() != "command_lifecycle" {
		t.Fatalf(
			"lifecycle migration = %d/%q, want 5/command_lifecycle",
			lifecycleMigration.Version(), lifecycleMigration.Name(),
		)
	}
	for _, required := range []string{
		"ADD COLUMN dispatched_at timestamptz",
		"ADD COLUMN acknowledged_at timestamptz",
		"'pending'",
		"'dispatched'",
		"'acknowledged'",
		"queue_control_commands_lifecycle_check",
	} {
		if !strings.Contains(lifecycleMigration.UpSQL(), required) {
			t.Fatalf("lifecycle UpSQL() missing %q", required)
		}
	}

	deadlineMigration := migrations[5]
	if deadlineMigration.Version() != 6 || deadlineMigration.Name() != "command_deadlines" {
		t.Fatalf(
			"deadline migration = %d/%q, want 6/command_deadlines",
			deadlineMigration.Version(), deadlineMigration.Name(),
		)
	}
	for _, required := range []string{
		"ADD COLUMN deadline timestamptz",
		"INTERVAL '30 seconds'",
		"queue_control_commands_deadline_check",
		"INTERVAL '5 minutes'",
	} {
		if !strings.Contains(deadlineMigration.UpSQL(), required) {
			t.Fatalf("deadline UpSQL() missing %q", required)
		}
	}

	authenticationMigration := migrations[6]
	if authenticationMigration.Version() != 7 ||
		authenticationMigration.Name() != "authentication_method" {
		t.Fatalf(
			"authentication migration = %d/%q, want 7/authentication_method",
			authenticationMigration.Version(), authenticationMigration.Name(),
		)
	}
	for _, required := range []string{
		"ADD COLUMN authentication_method varchar(256)",
		"SET authentication_method = 'legacy'",
		"queue_control_commands_authentication_method_check",
	} {
		if !strings.Contains(authenticationMigration.UpSQL(), required) {
			t.Fatalf("authentication UpSQL() missing %q", required)
		}
	}

	capabilityMigration := migrations[7]
	if capabilityMigration.Version() != 8 ||
		capabilityMigration.Name() != "capability_snapshots" {
		t.Fatalf(
			"capability migration = %d/%q, want 8/capability_snapshots",
			capabilityMigration.Version(), capabilityMigration.Name(),
		)
	}
	for _, required := range []string{
		"ADD COLUMN required_capability varchar(256)",
		"ADD COLUMN capability_available boolean",
		"ADD COLUMN acknowledged_by varchar(256)",
		"queue_control_commands_acknowledgement_check",
	} {
		if !strings.Contains(capabilityMigration.UpSQL(), required) {
			t.Fatalf("capability UpSQL() missing %q", required)
		}
	}

	ulidMigration := migrations[8]
	if ulidMigration.Version() != 9 ||
		ulidMigration.Name() != "ulid_command_identifiers" {
		t.Fatalf(
			"ULID migration = %d/%q, want 9/ulid_command_identifiers",
			ulidMigration.Version(), ulidMigration.Name(),
		)
	}
	for _, required := range []string{
		"ALTER COLUMN command_id TYPE text",
		"queue_control_commands_command_id_shape_check",
		"char_length(command_id) = 26",
		"command_id = lower(command_id)",
		"ALTER COLUMN command_id TYPE uuid",
	} {
		if !strings.Contains(ulidMigration.UpSQL()+ulidMigration.DownSQL(), required) {
			t.Fatalf("ULID migration missing %q", required)
		}
	}
}

// Applications fold these migrations into their own ledgers (Postal's
// migration 54 among them), so every checksum must match the published
// history byte for byte.
func TestMigrationSourceMatchesPublishedChecksums(t *testing.T) {
	t.Parallel()

	want := []struct {
		version  uint64
		name     string
		checksum string
	}{
		{1, "control_plane", "sha256:3b1fbf69322ecab2547744d506d2da2e7631efa033b33ca08f9841200d63d22b"},
		{2, "command_outcomes", "sha256:e9aa64d3f2cc5fc35a42f80e4629bb572af1bfb09229f28f6cfa35a481f1b2b6"},
		{3, "command_identifiers", "sha256:4f7d1b727ce3b19f694910cec06a4a56ed668bf91351347c7da228775a6eeccf"},
		{4, "sensitive_access_audit", "sha256:cde3cc2119e69e141730ef9f57e3570afc40e4aae744283eb00a42c547912e62"},
		{5, "command_lifecycle", "sha256:4d82a4e3a241d397fe2c8c7e42cd3f1ee03b0f6b59f73df258d41610327418af"},
		{6, "command_deadlines", "sha256:5a7c62ecbc6329d847a572223a8a51950cc2cd9e61e848ec6869cfa2d599f49f"},
		{7, "authentication_method", "sha256:e3263fa58af24deff3ed4fe45f3d44f2e1d7b6e14018fd634dbf062a37dfb533"},
		{8, "capability_snapshots", "sha256:577223bcfa465e16c852e3b38f97ffd0ace88ed0c928453f4241569f744f1ab1"},
		{9, "ulid_command_identifiers", "sha256:cf34bfdb9035f38560c6a3154f49609f94051a06c18d5bf8f7f1a07319a733e4"},
	}
	source, err := MigrationSource()
	if err != nil {
		t.Fatalf("MigrationSource() error = %v", err)
	}
	loaded, err := source.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded) != len(want) {
		t.Fatalf("len(Load()) = %d, want %d", len(loaded), len(want))
	}
	for index, migration := range loaded {
		if uint64(migration.Version()) != want[index].version || migration.Name() != want[index].name ||
			migration.TransactionMode() != migrations.TransactionModeDefault ||
			migration.DownSQL() == "" || migration.Checksum().String() != want[index].checksum {
			t.Fatalf("migration %d = %d/%s/%s, want %+v", index, migration.Version(), migration.Name(),
				migration.Checksum(), want[index])
		}
	}
}
