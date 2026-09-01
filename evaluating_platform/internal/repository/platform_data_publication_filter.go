package repository

import (
	"fmt"
	"strings"
)

func platformDataPublishedWhereClause(subTypeColumn, subType string, placeholderOffset int) (string, []interface{}) {
	conds := []string{"status = 'published'", "visibility = 'public'"}
	args := []interface{}{}

	if strings.TrimSpace(subType) != "" {
		args = append(args, subType)
		conds = append(conds, fmt.Sprintf("%s = $%d", subTypeColumn, placeholderOffset+len(args)))
	}

	return "WHERE " + strings.Join(conds, " AND "), args
}
