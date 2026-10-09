package usertimeentry

import "database/sql"

func isValidWorkLocation(loc *string) bool {
	return loc == nil || *loc == "office" || *loc == "remote"
}

func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// normalizeWorkLocation: у рабочего дня место всегда задано (по умолчанию офис), у остальных типов его нет
func normalizeWorkLocation(isWorkDay bool, loc sql.NullString) sql.NullString {
	if !isWorkDay {
		return sql.NullString{}
	}
	if !loc.Valid || loc.String == "" {
		return sql.NullString{String: "office", Valid: true}
	}
	return loc
}
