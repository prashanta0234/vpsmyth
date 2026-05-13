package db

import (
	"fmt"
	"net"
	"strings"
)

type WhitelistEntry struct {
	ID        int    `json:"id"`
	CIDR      string `json:"cidr"`
	Label     string `json:"label"`
	CreatedAt string `json:"created_at"`
}

func InitSecurityTables() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS ip_whitelist (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			cidr      TEXT NOT NULL UNIQUE,
			label     TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

func GetWhitelist() ([]WhitelistEntry, error) {
	rows, err := DB.Query(`SELECT id, cidr, label, created_at FROM ip_whitelist ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []WhitelistEntry
	for rows.Next() {
		var e WhitelistEntry
		if err := rows.Scan(&e.ID, &e.CIDR, &e.Label, &e.CreatedAt); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// NormalizeCIDR accepts a plain IP ("1.2.3.4") or CIDR ("1.2.3.4/24") and
// returns a canonical CIDR string, e.g. "1.2.3.4/32".
func NormalizeCIDR(input string) (string, error) {
	input = strings.TrimSpace(input)
	if strings.Contains(input, "/") {
		ip, network, err := net.ParseCIDR(input)
		if err != nil {
			return "", fmt.Errorf("invalid CIDR notation: %v", err)
		}
		// Return host/mask (not network address) so the stored value is readable
		bits, _ := network.Mask.Size()
		return fmt.Sprintf("%s/%d", ip.String(), bits), nil
	}
	ip := net.ParseIP(input)
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %s", input)
	}
	if ip.To4() != nil {
		return ip.String() + "/32", nil
	}
	return ip.String() + "/128", nil
}

func AddWhitelistEntry(cidr, label string) error {
	_, err := DB.Exec(
		`INSERT OR IGNORE INTO ip_whitelist (cidr, label) VALUES (?, ?)`,
		cidr, label,
	)
	return err
}

func DeleteWhitelistEntry(id int) error {
	_, err := DB.Exec(`DELETE FROM ip_whitelist WHERE id = ?`, id)
	return err
}

// IsIPAllowed returns true when the whitelist is empty (no restrictions) OR when
// the given IP matches at least one entry. Loopback is always allowed.
func IsIPAllowed(ipStr string) (bool, error) {
	entries, err := GetWhitelist()
	if err != nil {
		return false, err
	}
	if len(entries) == 0 {
		return true, nil
	}

	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return false, nil
	}
	if ip.IsLoopback() {
		return true, nil
	}

	for _, entry := range entries {
		_, network, err := net.ParseCIDR(entry.CIDR)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true, nil
		}
	}
	return false, nil
}
