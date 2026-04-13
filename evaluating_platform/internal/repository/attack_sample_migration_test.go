package repository

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"testing/quick"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/model"
	"evaluating_platform/pkg/db"
)

// migrationFiles lists all migrations including the fix (005).
// Migration 003 creates attack_samples with encrypted_path;
// Migration 005 renames encrypted_path → storage_path and drops unused columns.
var migrationFiles = []string{
	"../../migrations/001_init.sql",
	"../../migrations/002_chat.sql",
	"../../migrations/003_tool_categories.sql",
	"../../migrations/004_app_detector.sql",
	"../../migrations/005_fix_attack_samples_columns.sql",
}

// setupTestDB connects to a test PostgreSQL database and applies migrations 001-004.
// It returns the pool and a cleanup function that drops all tables.
// Requires TEST_DATABASE_URL environment variable (e.g. "postgres://postgres:postgres@localhost:5432/ep_test?sslmode=disable").
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := db.NewPostgresPool(ctx, dsn, 5, 1)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}

	// Clean up any previous state
	cleanupTables(t, pool)

	// Apply migrations 001-004 (unfixed schema)
	for _, mf := range migrationFiles {
		if err := db.MigrateUp(ctx, pool, mf); err != nil {
			t.Fatalf("apply migration %s: %v", mf, err)
		}
	}

	t.Cleanup(func() {
		cleanupTables(t, pool)
		pool.Close()
	})

	return pool
}

// cleanupTables drops all tables created by migrations to ensure a clean state.
func cleanupTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	// Drop in reverse dependency order
	tables := []string{
		"app_detectors",
		"eval_packages",
		"attack_samples",
		"templates",
		"billing_records",
		"balance_transactions",
		"audit_logs",
		"assessment_logs",
		"reports",
		"assessments",
		"target_systems",
		"assets",
		"users",
	}
	for _, tbl := range tables {
		_, _ = pool.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tbl))
	}
	// Drop the trigger function too
	_, _ = pool.Exec(ctx, "DROP FUNCTION IF EXISTS update_updated_at() CASCADE")
}

// createTestExpert inserts a minimal expert user and returns the user ID.
func createTestExpert(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	expertID := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, name, role, org_name)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		expertID,
		fmt.Sprintf("expert_%s@test.local", expertID.String()[:8]),
		"$2a$10$dummyhashfortest000000000000000000000000000000000000",
		"Test Expert",
		"expert",
		"Test Org",
	)
	if err != nil {
		t.Fatalf("create test expert: %v", err)
	}
	return expertID
}

// validSubTypes are the allowed sub_type values per the CHECK constraint.
var validSubTypes = []string{
	"direct_injection",
	"malicious_instruction",
	"compliance_detection",
	"malicious_poisoning",
}

// validStatuses are the allowed status values per the CHECK constraint.
var validStatuses = []string{"draft", "testing", "published", "deprecated"}

// validVisibilities are the allowed visibility values per the CHECK constraint.
var validVisibilities = []string{"private", "org", "public"}

// randomAttackSample generates a random AttackSample for property-based testing.
// It uses the provided expertID and random source to produce valid field values.
func randomAttackSample(expertID uuid.UUID, rng *rand.Rand) *model.AttackSample {
	return &model.AttackSample{
		ID:          uuid.New(),
		ExpertID:    expertID,
		SubType:     validSubTypes[rng.Intn(len(validSubTypes))],
		Name:        fmt.Sprintf("sample-%d", rng.Intn(100000)),
		Description: fmt.Sprintf("desc-%d", rng.Intn(100000)),
		StoragePath: fmt.Sprintf("samples/%s/%s.csv", expertID.String(), uuid.New().String()),
		FileHash:    fmt.Sprintf("%064x", rng.Int63()),
		SampleCount: rng.Intn(10000) + 1,
		FileSize:    int64(rng.Intn(10000000) + 100),
		Status:      validStatuses[rng.Intn(len(validStatuses))],
		Visibility:  validVisibilities[rng.Intn(len(validVisibilities))],
	}
}

// TestBugCondition_StoragePathColumnMismatch is a property-based exploration test
// that verifies all CRUD operations on AttackSampleRepository succeed.
//
// **Validates: Requirements 1.1, 1.2, 1.3, 1.4**
//
// On UNFIXED code (migrations 001-004 only), this test is EXPECTED TO FAIL because:
// - The DB table has column `encrypted_path` (from migration 003)
// - The Go code references column `storage_path` (in sampleSelectCols and all SQL queries)
// - PostgreSQL will return "column storage_path does not exist"
//
// After the fix (migration 005 renames encrypted_path → storage_path), this test should PASS.
func TestBugCondition_StoragePathColumnMismatch(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewAttackSampleRepository(pool)
	ctx := context.Background()
	expertID := createTestExpert(t, pool)

	// Property: For any randomly generated AttackSample, all CRUD operations
	// (Create, GetByID, ListByExpert, Delete) should succeed without
	// "column storage_path does not exist" errors.
	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))
		sample := randomAttackSample(expertID, rng)

		// 1. Create - INSERT INTO attack_samples (..., storage_path, ...)
		err := repo.Create(ctx, sample)
		if err != nil {
			if isStoragePathColumnError(err) {
				t.Logf("Create failed with column mismatch: %v", err)
			}
			return false
		}

		// 2. GetByID - SELECT ... storage_path ... FROM attack_samples WHERE id = $1
		got, err := repo.GetByID(ctx, sample.ID)
		if err != nil {
			if isStoragePathColumnError(err) {
				t.Logf("GetByID failed with column mismatch: %v", err)
			}
			return false
		}
		if got.StoragePath != sample.StoragePath {
			t.Logf("GetByID returned wrong StoragePath: got %q, want %q", got.StoragePath, sample.StoragePath)
			return false
		}

		// 3. ListByExpert - SELECT ... storage_path ... FROM attack_samples WHERE expert_id = $1
		samples, total, err := repo.ListByExpert(ctx, expertID, "", 100, 0)
		if err != nil {
			if isStoragePathColumnError(err) {
				t.Logf("ListByExpert failed with column mismatch: %v", err)
			}
			return false
		}
		if total < 1 || len(samples) < 1 {
			t.Logf("ListByExpert returned no results, expected at least 1")
			return false
		}

		// 4. Delete - DELETE ... RETURNING storage_path
		storagePath, err := repo.Delete(ctx, sample.ID, expertID)
		if err != nil {
			if isStoragePathColumnError(err) {
				t.Logf("Delete failed with column mismatch: %v", err)
			}
			return false
		}
		if storagePath != sample.StoragePath {
			t.Logf("Delete returned wrong storagePath: got %q, want %q", storagePath, sample.StoragePath)
			return false
		}

		return true
	}

	cfg := &quick.Config{
		MaxCount: 20, // 20 random samples is sufficient to demonstrate the bug
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Bug condition confirmed: CRUD operations fail due to column mismatch.\n"+
			"The DB has 'encrypted_path' but Go code references 'storage_path'.\n"+
			"Error: %v", err)
	}
}

// isStoragePathColumnError checks if the error is the specific PostgreSQL error
// about the storage_path column not existing.
func isStoragePathColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "storage_path") && strings.Contains(msg, "does not exist")
}

// ============================================================
// Preservation Property Tests (Task 2)
// ============================================================

// validTemplateSubTypes are the allowed sub_type values for templates.
var validTemplateSubTypes = []string{"role_play", "multilingual", "encoding_evasion"}

// attackSampleNonBuggyCols holds the column list for raw SQL operations
// that bypass the repository (which references storage_path).
// After migration 005, the column is named storage_path.
const attackSampleInsertNonBuggy = `INSERT INTO attack_samples
	(id, expert_id, sub_type, name, description, storage_path, file_hash,
	 sample_count, file_size, status, visibility)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`

const attackSampleSelectNonBuggy = `SELECT id, expert_id, sub_type, name, description,
	file_hash, sample_count, file_size, status, visibility, created_at, updated_at
	FROM attack_samples WHERE id = $1`

// TestPreservation_AttackSampleNonBuggyColumns verifies that all non-buggy columns
// in the attack_samples table are readable and writable via raw SQL on unfixed schema.
//
// **Validates: Requirements 3.4**
//
// On FIXED code (migrations 001-005), direct SQL using storage_path (the renamed
// DB column) should succeed. This confirms baseline behavior is preserved.
func TestPreservation_AttackSampleNonBuggyColumns(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	expertID := createTestExpert(t, pool)

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		id := uuid.New()
		subType := validSubTypes[rng.Intn(len(validSubTypes))]
		name := fmt.Sprintf("sample-%d", rng.Intn(100000))
		description := fmt.Sprintf("desc-%d", rng.Intn(100000))
		storagePath := fmt.Sprintf("samples/%s/%s.csv", expertID.String(), uuid.New().String())
		fileHash := fmt.Sprintf("%064x", rng.Int63())
		sampleCount := rng.Intn(10000) + 1
		fileSize := int64(rng.Intn(10000000) + 100)
		status := validStatuses[rng.Intn(len(validStatuses))]
		visibility := validVisibilities[rng.Intn(len(validVisibilities))]

		// INSERT using raw SQL with storage_path (renamed DB column)
		_, err := pool.Exec(ctx, attackSampleInsertNonBuggy,
			id, expertID, subType, name, description, storagePath, fileHash,
			sampleCount, fileSize, status, visibility,
		)
		if err != nil {
			t.Logf("Raw INSERT failed: %v", err)
			return false
		}

		// SELECT non-buggy columns
		var gotID uuid.UUID
		var gotExpertID uuid.UUID
		var gotSubType, gotName, gotDesc, gotHash, gotStatus, gotVis string
		var gotCount int
		var gotSize int64
		var gotCreated, gotUpdated interface{}

		err = pool.QueryRow(ctx, attackSampleSelectNonBuggy, id).Scan(
			&gotID, &gotExpertID, &gotSubType, &gotName, &gotDesc,
			&gotHash, &gotCount, &gotSize, &gotStatus, &gotVis,
			&gotCreated, &gotUpdated,
		)
		if err != nil {
			t.Logf("Raw SELECT failed: %v", err)
			return false
		}

		// Verify round-trip consistency for non-buggy columns
		if gotID != id {
			t.Logf("id mismatch: got %v, want %v", gotID, id)
			return false
		}
		if gotExpertID != expertID {
			t.Logf("expert_id mismatch: got %v, want %v", gotExpertID, expertID)
			return false
		}
		if gotSubType != subType {
			t.Logf("sub_type mismatch: got %q, want %q", gotSubType, subType)
			return false
		}
		if gotName != name {
			t.Logf("name mismatch: got %q, want %q", gotName, name)
			return false
		}
		if gotDesc != description {
			t.Logf("description mismatch: got %q, want %q", gotDesc, description)
			return false
		}
		if gotHash != fileHash {
			t.Logf("file_hash mismatch: got %q, want %q", gotHash, fileHash)
			return false
		}
		if gotCount != sampleCount {
			t.Logf("sample_count mismatch: got %d, want %d", gotCount, sampleCount)
			return false
		}
		if gotSize != fileSize {
			t.Logf("file_size mismatch: got %d, want %d", gotSize, fileSize)
			return false
		}
		if gotStatus != status {
			t.Logf("status mismatch: got %q, want %q", gotStatus, status)
			return false
		}
		if gotVis != visibility {
			t.Logf("visibility mismatch: got %q, want %q", gotVis, visibility)
			return false
		}

		// Clean up for next iteration
		_, _ = pool.Exec(ctx, "DELETE FROM attack_samples WHERE id = $1", id)
		return true
	}

	cfg := &quick.Config{MaxCount: 20}
	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Preservation FAILED: non-buggy columns are not readable/writable.\nError: %v", err)
	}
}

// TestPreservation_AttackSampleSubTypeFilter verifies that filtering by sub_type
// via raw SQL works correctly on unfixed schema.
//
// **Validates: Requirements 3.2**
func TestPreservation_AttackSampleSubTypeFilter(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	expertID := createTestExpert(t, pool)

	// Insert one sample per sub_type
	for _, st := range validSubTypes {
		id := uuid.New()
		_, err := pool.Exec(ctx, attackSampleInsertNonBuggy,
			id, expertID, st,
			fmt.Sprintf("sample-%s", st), "desc", "path/test.csv",
			"hash", 10, int64(100), "draft", "private",
		)
		if err != nil {
			t.Fatalf("Insert sample with sub_type %q: %v", st, err)
		}
	}

	// Verify filtering by each sub_type returns exactly 1 row
	for _, st := range validSubTypes {
		var count int
		err := pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM attack_samples WHERE expert_id = $1 AND sub_type = $2",
			expertID, st,
		).Scan(&count)
		if err != nil {
			t.Errorf("Count query for sub_type %q failed: %v", st, err)
			continue
		}
		if count != 1 {
			t.Errorf("Expected 1 sample with sub_type %q, got %d", st, count)
		}
	}
}

// TestPreservation_TemplatesCRUD verifies that the templates table CRUD operations
// work correctly on unfixed schema (unaffected table).
//
// **Validates: Requirements 3.5**
//
// The templates table should be completely unaffected by any attack_samples changes.
func TestPreservation_TemplatesCRUD(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	expertID := createTestExpert(t, pool)

	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		id := uuid.New()
		subType := validTemplateSubTypes[rng.Intn(len(validTemplateSubTypes))]
		name := fmt.Sprintf("template-%d", rng.Intn(100000))
		description := fmt.Sprintf("tmpl-desc-%d", rng.Intn(100000))
		content := fmt.Sprintf("You are a {{role}}. Seed: %d", rng.Int63())
		status := validStatuses[rng.Intn(len(validStatuses))]
		visibility := validVisibilities[rng.Intn(len(validVisibilities))]

		// CREATE
		_, err := pool.Exec(ctx,
			`INSERT INTO templates (id, expert_id, sub_type, name, description, content, status, visibility)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			id, expertID, subType, name, description, content, status, visibility,
		)
		if err != nil {
			t.Logf("Template INSERT failed: %v", err)
			return false
		}

		// READ
		var gotName, gotSubType, gotContent, gotStatus, gotVis string
		err = pool.QueryRow(ctx,
			`SELECT name, sub_type, content, status, visibility FROM templates WHERE id = $1`, id,
		).Scan(&gotName, &gotSubType, &gotContent, &gotStatus, &gotVis)
		if err != nil {
			t.Logf("Template SELECT failed: %v", err)
			return false
		}
		if gotName != name || gotSubType != subType || gotContent != content ||
			gotStatus != status || gotVis != visibility {
			t.Logf("Template read mismatch: got (%q,%q,%q,%q,%q) want (%q,%q,%q,%q,%q)",
				gotName, gotSubType, gotContent, gotStatus, gotVis,
				name, subType, content, status, visibility)
			return false
		}

		// UPDATE
		newName := fmt.Sprintf("updated-%d", rng.Intn(100000))
		_, err = pool.Exec(ctx,
			`UPDATE templates SET name = $1 WHERE id = $2`, newName, id,
		)
		if err != nil {
			t.Logf("Template UPDATE failed: %v", err)
			return false
		}
		var updatedName string
		err = pool.QueryRow(ctx, `SELECT name FROM templates WHERE id = $1`, id).Scan(&updatedName)
		if err != nil || updatedName != newName {
			t.Logf("Template UPDATE verify failed: err=%v, got=%q, want=%q", err, updatedName, newName)
			return false
		}

		// DELETE
		tag, err := pool.Exec(ctx, `DELETE FROM templates WHERE id = $1`, id)
		if err != nil {
			t.Logf("Template DELETE failed: %v", err)
			return false
		}
		if tag.RowsAffected() != 1 {
			t.Logf("Template DELETE affected %d rows, expected 1", tag.RowsAffected())
			return false
		}

		return true
	}

	cfg := &quick.Config{MaxCount: 20}
	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Preservation FAILED: templates table CRUD broken.\nError: %v", err)
	}
}

// TestPreservation_EvalPackagesTableUnaffected verifies that the eval_packages table
// structure is intact and CRUD operations work correctly on unfixed schema.
//
// **Validates: Requirements 3.5**
//
// The eval_packages table should be completely unaffected by any attack_samples changes.
func TestPreservation_EvalPackagesTableUnaffected(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	expertID := createTestExpert(t, pool)

	// validEvalPkgStatuses reuses the same status/visibility constraints as attack_samples.
	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		id := uuid.New()
		name := fmt.Sprintf("evalpkg-%d", rng.Intn(100000))
		description := fmt.Sprintf("pkg-desc-%d", rng.Intn(100000))
		encryptedPath := fmt.Sprintf("packages/%s/%s.espkg", expertID.String(), uuid.New().String())
		fileHash := fmt.Sprintf("%064x", rng.Int63())
		encryptionKeyID := fmt.Sprintf("key-%d", rng.Intn(10000))
		status := validStatuses[rng.Intn(len(validStatuses))]
		visibility := validVisibilities[rng.Intn(len(validVisibilities))]

		// CREATE
		_, err := pool.Exec(ctx,
			`INSERT INTO eval_packages
				(id, expert_id, name, description, encrypted_path, file_hash, encryption_key_id, status, visibility)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			id, expertID, name, description, encryptedPath, fileHash, encryptionKeyID, status, visibility,
		)
		if err != nil {
			t.Logf("EvalPackage INSERT failed: %v", err)
			return false
		}

		// READ
		var gotName, gotDesc, gotPath, gotHash, gotKeyID, gotStatus, gotVis string
		err = pool.QueryRow(ctx,
			`SELECT name, description, encrypted_path, file_hash, encryption_key_id, status, visibility
			 FROM eval_packages WHERE id = $1`, id,
		).Scan(&gotName, &gotDesc, &gotPath, &gotHash, &gotKeyID, &gotStatus, &gotVis)
		if err != nil {
			t.Logf("EvalPackage SELECT failed: %v", err)
			return false
		}
		if gotName != name || gotDesc != description || gotPath != encryptedPath ||
			gotHash != fileHash || gotKeyID != encryptionKeyID ||
			gotStatus != status || gotVis != visibility {
			t.Logf("EvalPackage read mismatch")
			return false
		}

		// UPDATE
		newName := fmt.Sprintf("updated-pkg-%d", rng.Intn(100000))
		_, err = pool.Exec(ctx,
			`UPDATE eval_packages SET name = $1 WHERE id = $2`, newName, id,
		)
		if err != nil {
			t.Logf("EvalPackage UPDATE failed: %v", err)
			return false
		}
		var updatedName string
		err = pool.QueryRow(ctx, `SELECT name FROM eval_packages WHERE id = $1`, id).Scan(&updatedName)
		if err != nil || updatedName != newName {
			t.Logf("EvalPackage UPDATE verify failed: err=%v, got=%q, want=%q", err, updatedName, newName)
			return false
		}

		// DELETE
		tag, err := pool.Exec(ctx, `DELETE FROM eval_packages WHERE id = $1`, id)
		if err != nil {
			t.Logf("EvalPackage DELETE failed: %v", err)
			return false
		}
		if tag.RowsAffected() != 1 {
			t.Logf("EvalPackage DELETE affected %d rows, expected 1", tag.RowsAffected())
			return false
		}

		return true
	}

	cfg := &quick.Config{MaxCount: 20}
	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Preservation FAILED: eval_packages table CRUD broken.\nError: %v", err)
	}
}
