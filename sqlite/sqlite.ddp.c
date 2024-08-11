#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/error.h"
#include "lib/sqlite3.h"
#include <assert.h>
#include <stdio.h>
#include <string.h>

static_assert(sizeof(int) == sizeof(ddpchar), "int must be 4 bytes");

typedef ddpint Zeiger;
typedef ddpint *ZeigerRef;

typedef Zeiger Datenbank;
typedef ZeigerRef DatenbankRef;

typedef Zeiger SQLAusdruck;
typedef ZeigerRef SQLAusdruckRef;

void Oeffne_DB(DatenbankRef p, ddpstring *db_name) {
	struct sqlite3 **db = (struct sqlite3 **)p;
	int err = sqlite3_open(db_name->str, db);
	if (err != SQLITE_OK) {
		ddp_error("Datenbank '%s' konnte nicht geöffnet werden: %s\n",
				  false, db_name->str, sqlite3_errmsg(*db));
		sqlite3_close(*db);
	}
}

void Schließe_DB(Datenbank db) {
	int err = sqlite3_close((struct sqlite3 *)db);
	if (err != SQLITE_OK) {
		ddp_error("Datenbank konnte nicht geschlossen werden: %s\n",
				  false, sqlite3_errmsg((struct sqlite3 *)db));
	}
}

void Statement_Vorbereiten(Datenbank pDB, SQLAusdruckRef pStmt,
						   ddpstring *sql) {
	struct sqlite3 *db = (struct sqlite3 *)pDB;
	struct sqlite3_stmt **stmt = (struct sqlite3_stmt **)pStmt;
	int err = sqlite3_prepare_v2(db, sql->str, -1, stmt, NULL);
	if (err != SQLITE_OK) {
		ddp_error("Fehler beim Vorbereiten des Statements: %s\n",
				  false, sqlite3_errmsg(db));
	}
}

void Statement_Zuruecksetzen(SQLAusdruck stmt) {
	int err = sqlite3_reset((struct sqlite3_stmt *)stmt);
	if (err != SQLITE_OK) {
		ddp_error("Fehler beim Zurücksetzen des Statements: %s\n",
				  false, sqlite3_errmsg(sqlite3_db_handle((struct sqlite3_stmt *)stmt)));
	}
}

void Statement_Schließen(SQLAusdruck stmt) {
	sqlite3_finalize((struct sqlite3_stmt *)stmt);
}

ddpbool Naechste_Zeile_Vorbereiten(SQLAusdruck stmt) {
	int err = sqlite3_step((struct sqlite3_stmt *)stmt) == SQLITE_ROW;
	if (err != SQLITE_DONE && err != SQLITE_ROW) {
		ddp_error("Fehler beim Ausführen des Statements: %s\n",
				  false, sqlite3_errmsg(sqlite3_db_handle((struct sqlite3_stmt *)stmt)));
	}
	return err == SQLITE_ROW;
}

ddpbool War_NULL(SQLAusdruck stmt, ddpint spalte) {
	return sqlite3_column_type((struct sqlite3_stmt *)stmt, spalte) == SQLITE_NULL;
}

void Lies_Spalte_Text(ddpstring *ret, SQLAusdruck stmt, ddpint spalte) {
	const unsigned char *text = sqlite3_column_text((struct sqlite3_stmt *)stmt, spalte);
	*ret = DDP_EMPTY_STRING;

	if (!text) {
		return;
	}

	int len = strlen((const char *)text);
	ret->cap = len + 1;
	ret->str = DDP_ALLOCATE(char, ret->cap);
	memcpy(ret->str, text, len);
}

ddpint Lies_Spalte_Zahl(SQLAusdruck stmt, ddpint spalte) {
	return sqlite3_column_int64((struct sqlite3_stmt *)stmt, spalte);
}

ddpfloat Lies_Spalte_Kommazahl(SQLAusdruck stmt, ddpint spalte) {
	return sqlite3_column_double((struct sqlite3_stmt *)stmt, spalte);
}

void Setze_Parameter_Null(SQLAusdruck stmt, ddpint i) {
	int err = sqlite3_bind_null((struct sqlite3_stmt *)stmt, (int)i);
	if (err != SQLITE_OK) {
		ddp_error("Fehler beim Setzen des Parameters " DDP_INT_FMT " auf NULL: %s\n",
				  false, i, sqlite3_errmsg(sqlite3_db_handle((struct sqlite3_stmt *)stmt)));
	}
}

void Setze_Parameter_Text(SQLAusdruck stmt, ddpint i, ddpstring *text) {
	int err = sqlite3_bind_text((struct sqlite3_stmt *)stmt, (int)i, text->str, -1, SQLITE_TRANSIENT);
	if (err != SQLITE_OK) {
		ddp_error("Fehler beim Setzen des Parameters " DDP_INT_FMT " auf einen Text: %s\n",
				  false, sqlite3_errmsg(sqlite3_db_handle((struct sqlite3_stmt *)stmt)));
	}
}

void Setze_Parameter_Zahl(SQLAusdruck stmt, ddpint i, ddpint zahl) {
	int err = sqlite3_bind_int64((struct sqlite3_stmt *)stmt, (int)i, zahl);
	if (err != SQLITE_OK) {
		ddp_error("Fehler beim Setzen des Parameters " DDP_INT_FMT " auf eine Zahl: %s\n",
				  false, sqlite3_errmsg(sqlite3_db_handle((struct sqlite3_stmt *)stmt)));
	}
}

void Setze_Parameter_Kommazahl(SQLAusdruck stmt, ddpint i, ddpfloat zahl) {
	int err = sqlite3_bind_double((struct sqlite3_stmt *)stmt, (int)i, zahl);
	if (err != SQLITE_OK) {
		ddp_error("Fehler beim Setzen des Parameters " DDP_INT_FMT " auf eine Kommazahl: %s\n",
				  false, sqlite3_errmsg(sqlite3_db_handle((struct sqlite3_stmt *)stmt)));
	}
}