package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	"maintify/core/pkg/rbac"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Initialize database connection
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	rbacService := rbac.NewPostgreSQLRBACService(db)

	switch command {
	case "migrate":
		runMigrations(db)
	case "create-admin":
		createSystemAdmin(rbacService)
	case "create-org":
		createOrganization(rbacService)
	case "list-users":
		listUsers(rbacService)
	case "list-orgs":
		listOrganizations(rbacService)
	case "migration-status":
		migrationStatus(db)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Maintify RBAC CLI")
	fmt.Println("Usage:")
	fmt.Println("  rbac-cli migrate              - Run database migrations")
	fmt.Println("  rbac-cli create-admin          - Create system administrator")
	fmt.Println("  rbac-cli create-org            - Create organization")
	fmt.Println("  rbac-cli list-users            - List all users")
	fmt.Println("  rbac-cli list-orgs             - List all organizations")
	fmt.Println("  rbac-cli migration-status      - Show migration status")
	fmt.Println("")
	fmt.Println("Environment variables:")
	fmt.Println("  DB_HOST          - Database host (default: localhost)")
	fmt.Println("  DB_PORT          - Database port (default: 5432)")
	fmt.Println("  DB_USER          - Database user (default: maintify)")
	fmt.Println("  DB_PASSWORD      - Database password (default: maintify)")
	fmt.Println("  DB_NAME          - Database name (default: maintify_core)")
	fmt.Println("")
	fmt.Println("Non-interactive admin creation (all must be set):")
	fmt.Println("  ADMIN_EMAIL      - Admin email address")
	fmt.Println("  ADMIN_USERNAME   - Admin username")
	fmt.Println("  ADMIN_PASSWORD   - Admin password (min 8 chars)")
	fmt.Println("  ADMIN_FIRST_NAME - Admin first name")
	fmt.Println("  ADMIN_LAST_NAME  - Admin last name")
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func initDatabase() (*sqlx.DB, error) {
	dbHost := getEnvWithDefault("DB_HOST", "localhost")
	dbPort := getEnvWithDefault("DB_PORT", "5432")
	dbUser := getEnvWithDefault("DB_USER", "maintify")
	dbPassword := getEnvWithDefault("DB_PASSWORD", "maintify")
	dbName := getEnvWithDefault("DB_NAME", "maintify_core")
	dbSSLMode := getEnvWithDefault("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func getMigrationsDir() string {
	if dir := os.Getenv("MIGRATIONS_DIR"); dir != "" {
		return dir
	}
	if _, err := os.Stat("core/migrations"); err == nil {
		return "core/migrations"
	}
	if _, err := os.Stat("migrations"); err == nil {
		return "migrations"
	}
	return "core/migrations"
}

func runMigrations(db *sqlx.DB) {
	migrationService := rbac.NewMigrationService(db, getMigrationsDir())
	err := migrationService.ApplyMigrations()
	if err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}
	fmt.Println("Migrations applied successfully")
}

func migrationStatus(db *sqlx.DB) {
	migrationService := rbac.NewMigrationService(db, getMigrationsDir())
	migrations, err := migrationService.GetMigrationStatus()
	if err != nil {
		log.Fatalf("Failed to get migration status: %v", err)
	}

	fmt.Println("Migration Status:")
	fmt.Println("=================")
	for _, migration := range migrations {
		status := "Not Applied"
		if migration.Applied {
			status = "Applied"
		}
		fmt.Printf("%03d %-40s %s\n", migration.ID, migration.Name, status)
	}
}

func createSystemAdmin(rbacService rbac.RBACService) {
	fmt.Println("Creating System Administrator")
	fmt.Println("=============================")

	var email, username, firstName, lastName, password string

	// Non-interactive mode: all ADMIN_* env vars must be set
	envEmail := os.Getenv("ADMIN_EMAIL")
	envUsername := os.Getenv("ADMIN_USERNAME")
	envPassword := os.Getenv("ADMIN_PASSWORD")
	envFirstName := os.Getenv("ADMIN_FIRST_NAME")
	envLastName := os.Getenv("ADMIN_LAST_NAME")

	if envEmail != "" && envUsername != "" && envPassword != "" && envFirstName != "" && envLastName != "" {
		email = envEmail
		username = envUsername
		password = envPassword
		firstName = envFirstName
		lastName = envLastName
		fmt.Println("(non-interactive mode — using ADMIN_* environment variables)")
	} else {
		// Interactive mode
		fmt.Print("Email: ")
		fmt.Scanln(&email)

		fmt.Print("Username: ")
		fmt.Scanln(&username)

		fmt.Print("First Name: ")
		fmt.Scanln(&firstName)

		fmt.Print("Last Name: ")
		fmt.Scanln(&lastName)

		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("Failed to read password: %v", err)
		}
		password = string(passwordBytes)
		fmt.Println()

		fmt.Print("Confirm Password: ")
		confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("Failed to read password confirmation: %v", err)
		}
		if password != string(confirmBytes) {
			log.Fatal("Passwords do not match")
		}
		fmt.Println()
	}

	if len(password) < 8 {
		log.Fatal("Password must be at least 8 characters long")
	}

	// Create user
	user := &rbac.User{
		Email:         email,
		Username:      username,
		PasswordHash:  password, // Pass raw password, service will hash it
		FirstName:     firstName,
		LastName:      lastName,
		IsActive:      true,
		IsSystemAdmin: true,
		Metadata:      make(map[string]interface{}),
	}

	ctx := context.Background()
	err := rbacService.CreateUser(ctx, user)
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	// Get system organization
	systemOrg, err := rbacService.GetOrganizationBySlug(ctx, "system")
	if err != nil {
		log.Fatalf("Failed to get system organization: %v", err)
	}

	// Get system admin role
	roles, err := rbacService.ListRoles(ctx, systemOrg.ID, 10, 0)
	if err != nil {
		log.Fatalf("Failed to get roles: %v", err)
	}

	var systemAdminRole *rbac.Role
	for _, role := range roles {
		if role.Name == "system-admin" {
			systemAdminRole = role
			break
		}
	}

	if systemAdminRole == nil {
		log.Fatal("System admin role not found")
	}

	// Assign system admin role to user
	assignment := &rbac.UserRoleAssignment{
		UserID:           user.ID,
		RoleID:           systemAdminRole.ID,
		OrganizationID:   systemOrg.ID,
		ValidFrom:        time.Now(),
		AssignmentReason: "Initial system administrator setup",
		IsActive:         true,
	}

	err = rbacService.AssignRoleToUser(ctx, assignment)
	if err != nil {
		log.Fatalf("Failed to assign role to user: %v", err)
	}

	fmt.Printf("   System administrator created successfully!\n")
	fmt.Printf("   User ID: %s\n", user.ID)
	fmt.Printf("   Email: %s\n", user.Email)
	fmt.Printf("   Username: %s\n", user.Username)
}

func createOrganization(rbacService rbac.RBACService) {
	fmt.Println("Creating Organization")
	fmt.Println("====================")

	fmt.Print("Name: ")
	var name string
	fmt.Scanln(&name)

	fmt.Print("Slug (URL-friendly identifier): ")
	var slug string
	fmt.Scanln(&slug)

	fmt.Print("Description: ")
	var description string
	fmt.Scanln(&description)

	// Create organization
	org := &rbac.Organization{
		Name:        name,
		Slug:        slug,
		Description: description,
		Settings:    make(map[string]interface{}),
	}

	ctx := context.Background()
	err := rbacService.CreateOrganization(ctx, org)
	if err != nil {
		log.Fatalf("Failed to create organization: %v", err)
	}

	fmt.Printf("   Organization created successfully!\n")
	fmt.Printf("   ID: %s\n", org.ID)
	fmt.Printf("   Name: %s\n", org.Name)
	fmt.Printf("   Slug: %s\n", org.Slug)
}

func listUsers(rbacService rbac.RBACService) {
	ctx := context.Background()
	users, err := rbacService.ListUsers(ctx, nil, 100, 0)
	if err != nil {
		log.Fatalf("Failed to list users: %v", err)
	}

	fmt.Println("Users")
	fmt.Println("=====")
	if len(users) == 0 {
		fmt.Println("No users found")
		return
	}

	fmt.Printf("%-36s %-20s %-20s %-10s %-10s\n", "ID", "Email", "Username", "Active", "SysAdmin")
	fmt.Println(strings.Repeat("-", 100))
	for _, user := range users {
		active := "No"
		if user.IsActive {
			active = "Yes"
		}
		sysAdmin := "No"
		if user.IsSystemAdmin {
			sysAdmin = "Yes"
		}
		fmt.Printf("%-36s %-20s %-20s %-10s %-10s\n",
			user.ID, user.Email, user.Username, active, sysAdmin)
	}
}

func listOrganizations(rbacService rbac.RBACService) {
	ctx := context.Background()
	orgs, err := rbacService.ListOrganizations(ctx, 100, 0)
	if err != nil {
		log.Fatalf("Failed to list organizations: %v", err)
	}

	fmt.Println("Organizations")
	fmt.Println("=============")
	if len(orgs) == 0 {
		fmt.Println("No organizations found")
		return
	}

	fmt.Printf("%-36s %-20s %-20s %-50s\n", "ID", "Name", "Slug", "Description")
	fmt.Println(strings.Repeat("-", 130))
	for _, org := range orgs {
		fmt.Printf("%-36s %-20s %-20s %-50s\n",
			org.ID, org.Name, org.Slug, org.Description)
	}
}
