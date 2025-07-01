// backend/cmd/seed/main.go
package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	// "net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Ayash-Bera/ophelia/backend/internal/alchemyst"
	"github.com/Ayash-Bera/ophelia/backend/internal/config"
	"github.com/Ayash-Bera/ophelia/backend/internal/database"
	"github.com/Ayash-Bera/ophelia/backend/internal/models"
	"github.com/Ayash-Bera/ophelia/backend/internal/repository"
	"github.com/Ayash-Bera/ophelia/backend/pkg/utils"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// WikiPageConfig represents configuration for a wiki page
type WikiPageConfig struct {
	Title    string
	URL      string
	Priority int
	Sections []string
}

// WikiSection represents a section of a wiki page
type WikiSection struct {
	Title   string
	Content string
	Anchor  string
	Level   int
}

// ContentSeeder handles wiki content scraping and seeding
type ContentSeeder struct {
	collector        *colly.Collector
	alchemystService *alchemyst.Service
	repoManager      *repository.RepositoryManager
	logger           *logrus.Logger
	processed        map[string]bool
	errors           []error
}

var (
	// High-priority Arch Wiki pages with common troubleshooting content
	ArchWikiPages = []WikiPageConfig{
		// Core troubleshooting (Priority 10-9)
		{Title: "General_troubleshooting", Priority: 10, URL: "https://wiki.archlinux.org/title/General_troubleshooting"},
		{Title: "Installation_guide", Priority: 10, URL: "https://wiki.archlinux.org/title/Installation_guide"},
		{Title: "System_maintenance", Priority: 9, URL: "https://wiki.archlinux.org/title/System_maintenance"},

		// Package management (Priority 9-8)
		{Title: "Pacman", Priority: 9, URL: "https://wiki.archlinux.org/title/Pacman"},
		// {Title: "Pacman_troubleshooting", Priority: 9, URL: "https://wiki.archlinux.org/title/Pacman/Troubleshooting"},
		{Title: "AUR", Priority: 8, URL: "https://wiki.archlinux.org/title/Arch_User_Repository"},
		{Title: "makepkg", Priority: 8, URL: "https://wiki.archlinux.org/title/Makepkg"},

		// Network (Priority 8-7)
		{Title: "NetworkManager", Priority: 8, URL: "https://wiki.archlinux.org/title/NetworkManager"},
		{Title: "Network_configuration", Priority: 7, URL: "https://wiki.archlinux.org/title/Network_configuration"},
		{Title: "Wireless_network_configuration", Priority: 7, URL: "https://wiki.archlinux.org/title/Wireless_network_configuration"},
		{Title: "OpenVPN", Priority: 6, URL: "https://wiki.archlinux.org/title/OpenVPN"},

		// Graphics (Priority 8-6)
		{Title: "Xorg", Priority: 8, URL: "https://wiki.archlinux.org/title/Xorg"},
		{Title: "NVIDIA", Priority: 7, URL: "https://wiki.archlinux.org/title/NVIDIA"},
		{Title: "NVIDIA_troubleshooting", Priority: 7, URL: "https://wiki.archlinux.org/title/NVIDIA/Troubleshooting"},
		{Title: "AMDGPU", Priority: 7, URL: "https://wiki.archlinux.org/title/AMDGPU"},
		{Title: "Intel_graphics", Priority: 6, URL: "https://wiki.archlinux.org/title/Intel_graphics"},
		{Title: "Wayland", Priority: 6, URL: "https://wiki.archlinux.org/title/Wayland"},

		// Audio (Priority 7-6)
		{Title: "Advanced_Linux_Sound_Architecture", Priority: 7, URL: "https://wiki.archlinux.org/title/Advanced_Linux_Sound_Architecture"},
		{Title: "PulseAudio", Priority: 6, URL: "https://wiki.archlinux.org/title/PulseAudio"},
		{Title: "PulseAudio_troubleshooting", Priority: 6, URL: "https://wiki.archlinux.org/title/PulseAudio/Troubleshooting"},
		{Title: "PipeWire", Priority: 6, URL: "https://wiki.archlinux.org/title/PipeWire"},

		// Boot/System (Priority 7-6)
		{Title: "GRUB", Priority: 7, URL: "https://wiki.archlinux.org/title/GRUB"},
		{Title: "Systemd", Priority: 7, URL: "https://wiki.archlinux.org/title/Systemd"},
		{Title: "Kernel_parameters", Priority: 6, URL: "https://wiki.archlinux.org/title/Kernel_parameters"},
		{Title: "Fstab", Priority: 6, URL: "https://wiki.archlinux.org/title/Fstab"},
		{Title: "Arch_boot_process", Priority: 6, URL: "https://wiki.archlinux.org/title/Arch_boot_process"},

		// Hardware (Priority 6-5)
		{Title: "Bluetooth", Priority: 6, URL: "https://wiki.archlinux.org/title/Bluetooth"},
		{Title: "Power_management", Priority: 5, URL: "https://wiki.archlinux.org/title/Power_management"},
		{Title: "Laptop", Priority: 5, URL: "https://wiki.archlinux.org/title/Laptop"},
		{Title: "Hardware_video_acceleration", Priority: 5, URL: "https://wiki.archlinux.org/title/Hardware_video_acceleration"},

		// Desktop Environments (Priority 6-5)
		{Title: "GNOME", Priority: 6, URL: "https://wiki.archlinux.org/title/GNOME"},
		{Title: "GNOME_troubleshooting", Priority: 6, URL: "https://wiki.archlinux.org/title/GNOME/Troubleshooting"},
		{Title: "KDE", Priority: 5, URL: "https://wiki.archlinux.org/title/KDE"},
		{Title: "Xfce", Priority: 5, URL: "https://wiki.archlinux.org/title/Xfce"},

		// Gaming (Priority 5-4)
		{Title: "Steam", Priority: 5, URL: "https://wiki.archlinux.org/title/Steam"},
		{Title: "Steam_troubleshooting", Priority: 5, URL: "https://wiki.archlinux.org/title/Steam/Troubleshooting"},
		{Title: "Gaming", Priority: 4, URL: "https://wiki.archlinux.org/title/Gaming"},

		// Services & Virtualization (Priority 5-4)
		{Title: "OpenSSH", Priority: 5, URL: "https://wiki.archlinux.org/title/OpenSSH"},
		{Title: "Docker", Priority: 4, URL: "https://wiki.archlinux.org/title/Docker"},
		{Title: "VirtualBox", Priority: 4, URL: "https://wiki.archlinux.org/title/VirtualBox"},

		// Printing & Multimedia (Priority 4-3)
		{Title: "CUPS", Priority: 4, URL: "https://wiki.archlinux.org/title/CUPS"},
		{Title: "CUPS_troubleshooting", Priority: 4, URL: "https://wiki.archlinux.org/title/CUPS/Troubleshooting"},
		{Title: "Firefox", Priority: 3, URL: "https://wiki.archlinux.org/title/Firefox"},
		{Title: "Chromium", Priority: 3, URL: "https://wiki.archlinux.org/title/Chromium"},

		// File Systems & Storage (Priority 4-3)
		{Title: "File_systems", Priority: 4, URL: "https://wiki.archlinux.org/title/File_systems"},
		{Title: "USB_storage_devices", Priority: 3, URL: "https://wiki.archlinux.org/title/USB_storage_devices"},
		{Title: "Solid_state_drive", Priority: 3, URL: "https://wiki.archlinux.org/title/Solid_state_drive"},
	}

	// Command line flags
	dryRun     = flag.Bool("dry-run", false, "Don't upload to Alchemyst, just print what would be uploaded")
	verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	pageLimit  = flag.Int("limit", 0, "Limit number of pages to process (0 = all)")
	concurrent = flag.Int("concurrent", 2, "Number of concurrent requests")
	delay      = flag.Duration("delay", 2*time.Second, "Delay between requests")
)

func main() {
	flag.Parse()

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found: %v", err)
	}

	// Initialize logger
	logger := utils.GetLogger()
	if *verbose {
		logger.SetLevel(logrus.DebugLevel)
	}

	logger.Info("Starting Arch Wiki content seeder...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	var alchemystService *alchemyst.Service
	var repoManager *repository.RepositoryManager

	if !*dryRun {
		// Validate Alchemyst configuration
		if err := cfg.ValidateAlchemyst(); err != nil {
			logger.WithError(err).Fatal("Alchemyst configuration validation failed")
		}

		// Initialize database for tracking
		dbConfig := &database.Config{
			DatabaseURL: cfg.Database.URL,
			RedisURL:    cfg.Redis.URL,
			LogLevel:    os.Getenv("LOG_LEVEL"),
		}

		dbManager, err := database.NewManager(dbConfig, logger)
		if err != nil {
			logger.WithError(err).Fatal("Failed to initialize database manager")
		}
		defer dbManager.Close()

		repoManager = repository.NewRepositoryManager(dbManager.DB)

		// Initialize Alchemyst client and service
		alchemystClient := alchemyst.NewClient(cfg.Alchemyst.BaseURL, cfg.Alchemyst.APIKey, logger)
		alchemystService = alchemyst.NewService(alchemystClient, logger)
	}

	// Create content seeder
	seeder := NewContentSeeder(alchemystService, repoManager, logger)

	// Process pages
	ctx := context.Background()
	if err := seeder.SeedContent(ctx); err != nil {
		logger.WithError(err).Fatal("Content seeding failed")
	}

	logger.Info("Content seeding completed successfully!")
}

func NewContentSeeder(alchemystService *alchemyst.Service, repoManager *repository.RepositoryManager, logger *logrus.Logger) *ContentSeeder {
	// Configure Colly collector
	c := colly.NewCollector(
		colly.UserAgent("ArchSearch-Bot/1.0 (+https://github.com/yourusername/arch-search)"),
	)

	// Enable debug mode if verbose (remove debugger due to compatibility issues)
	// if *verbose {
	// 	c.Debugger = &debug.LogDebugger{}
	// }

	// Configure limits and delays
	c.Limit(&colly.LimitRule{
		DomainGlob:  "wiki.archlinux.org",
		Parallelism: *concurrent,
		Delay:       *delay,
	})

	// Configure timeouts
	c.SetRequestTimeout(30 * time.Second)

	return &ContentSeeder{
		collector:        c,
		alchemystService: alchemystService,
		repoManager:      repoManager,
		logger:           logger,
		processed:        make(map[string]bool),
		errors:           make([]error, 0),
	}
}

func (cs *ContentSeeder) SeedContent(ctx context.Context) error {
	cs.logger.Info("Starting content seeding process...")

	// Sort pages by priority
	pages := make([]WikiPageConfig, len(ArchWikiPages))
	copy(pages, ArchWikiPages)

	// Sort by priority (descending) - using a simple bubble sort for clarity
	for i := 0; i < len(pages)-1; i++ {
		for j := i + 1; j < len(pages); j++ {
			if pages[i].Priority < pages[j].Priority {
				pages[i], pages[j] = pages[j], pages[i]
			}
		}
	}

	// Apply page limit if specified
	if *pageLimit > 0 && *pageLimit < len(pages) {
		pages = pages[:*pageLimit]
		cs.logger.WithField("limit", *pageLimit).Info("Limited pages to process")
	}

	cs.logger.WithField("total_pages", len(pages)).Info("Processing wiki pages")

	// Process each page
	for i, page := range pages {
		cs.logger.WithFields(logrus.Fields{
			"page":     page.Title,
			"priority": page.Priority,
			"progress": fmt.Sprintf("%d/%d", i+1, len(pages)),
		}).Info("Processing page")

		if err := cs.processPage(ctx, page); err != nil {
			cs.logger.WithError(err).WithField("page", page.Title).Error("Failed to process page")
			cs.errors = append(cs.errors, fmt.Errorf("failed to process %s: %w", page.Title, err))
			continue
		}

		cs.processed[page.Title] = true
		cs.logger.WithField("page", page.Title).Info("Page processed successfully")

		// Small delay between pages
		time.Sleep(500 * time.Millisecond)
	}

	// Report results
	cs.logger.WithFields(logrus.Fields{
		"processed": len(cs.processed),
		"errors":    len(cs.errors),
	}).Info("Content seeding completed")

	if len(cs.errors) > 0 {
		cs.logger.Warn("Some pages failed to process:")
		for _, err := range cs.errors {
			cs.logger.WithError(err).Warn("Processing error")
		}
	}

	return nil
}

// Fix in cmd/seed/main.go - processPage function

func (cs *ContentSeeder) processPage(ctx context.Context, page WikiPageConfig) error {
	var content string
	var extractedSections []WikiSection
	var processingError error

	// Create a new collector for each page to avoid state issues
	c := colly.NewCollector(
		colly.UserAgent("ArchSearch-Bot/1.0 (+https://github.com/yourusername/arch-search)"),
	)

	// Configure limits and delays
	c.Limit(&colly.LimitRule{
		DomainGlob:  "wiki.archlinux.org",
		Parallelism: 1, // Use 1 for individual page processing
		Delay:       *delay,
	})

	c.SetRequestTimeout(30 * time.Second)

	// Configure collector for this specific page
	c.OnHTML("#mw-content-text", func(e *colly.HTMLElement) {
		// Extract main content
		content = cs.extractPageContent(e)

		// Extract sections
		extractedSections = cs.extractSections(e, page.Title)

		cs.logger.WithFields(logrus.Fields{
			"page":           page.Title,
			"content_length": len(content),
			"sections":       len(extractedSections),
		}).Debug("Content extracted")
	})

	c.OnError(func(r *colly.Response, err error) {
		processingError = err
	})

	// Visit the page
	err := c.Visit(page.URL)
	if err != nil {
		return fmt.Errorf("failed to visit page: %w", err)
	}

	if processingError != nil {
		return fmt.Errorf("processing error: %w", processingError)
	}

	if content == "" {
		return fmt.Errorf("no content extracted from page")
	}

	// Rest of the function remains the same...
	errorPatterns := cs.extractErrorPatterns(content)
	contentHash := cs.createContentHash(content)

	if !*dryRun && cs.repoManager != nil {
		if err := cs.updateContentMetadata(page, contentHash, errorPatterns, len(extractedSections), content); err != nil {
			cs.logger.WithError(err).Warn("Failed to update content metadata")
		}
	}

	if *dryRun {
		cs.logger.WithFields(logrus.Fields{
			"page":           page.Title,
			"content_length": len(content),
			"sections":       len(extractedSections),
			"error_patterns": len(errorPatterns),
			"hash":           contentHash[:8],
		}).Info("DRY RUN: Would upload content")
		return nil
	}

	// Upload main content to Alchemyst
	if err := cs.uploadToAlchemyst(ctx, page.Title, content, page.URL); err != nil {
		return fmt.Errorf("failed to upload main content: %w", err)
	}

	// Upload sections separately for better search granularity
	for i, section := range extractedSections {
		sectionTitle := fmt.Sprintf("%s/%s", page.Title, section.Title)
		if err := cs.uploadToAlchemyst(ctx, sectionTitle, section.Content, page.URL+"#"+section.Anchor); err != nil {
			cs.logger.WithError(err).WithField("section", sectionTitle).Warn("Failed to upload section")
			continue
		}

		// Log progress for long pages
		if len(extractedSections) > 10 && i%5 == 0 {
			cs.logger.WithFields(logrus.Fields{
				"page":     page.Title,
				"progress": fmt.Sprintf("%d/%d", i+1, len(extractedSections)),
			}).Debug("Section upload progress")
		}
	}

	return nil
}

func (cs *ContentSeeder) extractPageContent(e *colly.HTMLElement) string {
	// Remove unwanted elements
	e.DOM.Find(".navbox, .infobox, .ambox, .toc, .printfooter, .catlinks").Remove()
	e.DOM.Find("#toc, .noprint, .editlink, .mw-editsection").Remove()

	// Get text content
	text := strings.TrimSpace(e.DOM.Text())

	// Clean up whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(text, "\n\n")

	return text
}

func (cs *ContentSeeder) extractSections(e *colly.HTMLElement, pageTitle string) []WikiSection {
	var sections []WikiSection

	e.DOM.Find("h2, h3, h4").Each(func(i int, selection *goquery.Selection) {
		// Get section title
		titleText := strings.TrimSpace(selection.Find(".mw-headline").Text())
		if titleText == "" {
			return
		}

		// Get anchor
		anchor := ""
		if id, exists := selection.Find(".mw-headline").Attr("id"); exists {
			anchor = id
		}

		// Get section level based on tag name
		tagName := goquery.NodeName(selection)
		level := 2 // default
		switch tagName {
		case "h2":
			level = 2
		case "h3":
			level = 3
		case "h4":
			level = 4
		}

		// Get section content (find content until next heading)
		var content strings.Builder

		// Navigate through siblings until we hit another heading
		selection.NextUntil("h2, h3, h4").Each(func(j int, sibling *goquery.Selection) {
			// Skip certain elements
			if sibling.Is("table") || sibling.HasClass("navbox") || sibling.HasClass("ambox") {
				return
			}

			text := strings.TrimSpace(sibling.Text())
			if text != "" {
				content.WriteString(text + "\n")
			}
		})

		sectionContent := strings.TrimSpace(content.String())

		// Only include sections with substantial content
		if len(sectionContent) > 50 {
			sections = append(sections, WikiSection{
				Title:   titleText,
				Content: sectionContent,
				Anchor:  anchor,
				Level:   level,
			})
		}
	})

	cs.logger.WithFields(logrus.Fields{
		"page":     pageTitle,
		"sections": len(sections),
	}).Debug("Extracted sections")

	return sections
}

func (cs *ContentSeeder) extractErrorPatterns(content string) []string {
	patterns := make(map[string]bool)

	// Common error patterns in Arch Linux
	errorRegexes := []*regexp.Regexp{
		regexp.MustCompile(`(?i)error[:\s]+[a-zA-Z0-9\s\-\._/]+`),
		regexp.MustCompile(`(?i)failed[:\s]+[a-zA-Z0-9\s\-\._/]+`),
		regexp.MustCompile(`(?i)cannot[:\s]+[a-zA-Z0-9\s\-\._/]+`),
		regexp.MustCompile(`(?i)unable to[:\s]+[a-zA-Z0-9\s\-\._/]+`),
		regexp.MustCompile(`(?i)permission denied[:\s]*[a-zA-Z0-9\s\-\._/]*`),
		regexp.MustCompile(`(?i)no such file or directory[:\s]*[a-zA-Z0-9\s\-\._/]*`),
		regexp.MustCompile(`(?i)command not found[:\s]*[a-zA-Z0-9\s\-\._/]*`),
		regexp.MustCompile(`(?i)segmentation fault[:\s]*[a-zA-Z0-9\s\-\._/]*`),
		regexp.MustCompile(`(?i)kernel panic[:\s]*[a-zA-Z0-9\s\-\._/]*`),
		regexp.MustCompile(`(?i)dependency.*conflict[:\s]*[a-zA-Z0-9\s\-\._/]*`),
		regexp.MustCompile(`(?i)package.*not found[:\s]*[a-zA-Z0-9\s\-\._/]*`),
		regexp.MustCompile(`(?i)service.*failed[:\s]*[a-zA-Z0-9\s\-\._/]*`),
	}

	// Extract patterns
	for _, regex := range errorRegexes {
		matches := regex.FindAllString(content, -1)
		for _, match := range matches {
			// Clean and normalize the pattern
			pattern := strings.TrimSpace(match)
			pattern = regexp.MustCompile(`\s+`).ReplaceAllString(pattern, " ")

			if len(pattern) > 5 && len(pattern) < 100 {
				patterns[strings.ToLower(pattern)] = true
			}
		}
	}

	// Convert to slice
	var result []string
	for pattern := range patterns {
		result = append(result, pattern)
	}

	return result
}

func (cs *ContentSeeder) createContentHash(content string) string {
	hash := md5.Sum([]byte(content))
	return hex.EncodeToString(hash[:])
}

func (cs *ContentSeeder) updateContentMetadata(page WikiPageConfig, contentHash string, errorPatterns []string, sectionCount int, content string) error {
	// Convert string slice to StringArray
	var patterns models.StringArray = errorPatterns

	// Get current time
	now := time.Now()

	contentMetadata := &models.ContentMetadata{
		WikiPageTitle: page.Title,
		ContentHash:   contentHash,
		PageURL:       page.URL,
		ErrorPatterns: patterns,
		WordCount:     cs.estimateWordCount(content),
		SectionCount:  sectionCount,
		LastCrawled:   &now,
		CrawlStatus:   "completed",
		IsActive:      true,
	}

	// Try to update existing record first
	existing, err := cs.repoManager.ContentMetadata.GetByTitle(page.Title)
	if err == nil {
		// Update existing
		existing.ContentHash = contentHash
		existing.ErrorPatterns = patterns
		existing.WordCount = cs.estimateWordCount(content)
		existing.SectionCount = sectionCount
		existing.LastCrawled = &now
		existing.CrawlStatus = "completed"

		return cs.repoManager.ContentMetadata.Update(existing)
	}

	// Create new record
	return cs.repoManager.ContentMetadata.Create(contentMetadata)
}

func (cs *ContentSeeder) estimateWordCount(content string) int {
	// Simple word counting
	words := strings.Fields(content)
	return len(words)
}

func (cs *ContentSeeder) uploadToAlchemyst(ctx context.Context, title, content, wikiURL string) error {
	if cs.alchemystService == nil {
		return fmt.Errorf("alchemyst service not initialized")
	}

	cs.logger.WithFields(logrus.Fields{
		"title":          title,
		"content_length": len(content),
		"url":            wikiURL,
	}).Debug("Uploading to Alchemyst")

	return cs.alchemystService.AddWikiContent(ctx, title, content, wikiURL)
}
