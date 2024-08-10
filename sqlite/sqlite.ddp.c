#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "lib/sqlite3.h"
#include <stdio.h>
#include <string.h>

typedef ddpint Zeiger;
typedef ddpint *ZeigerRef;

typedef Zeiger Datenbank;
typedef ZeigerRef DatenbankRef;

typedef Zeiger SQLAusdruck;
typedef ZeigerRef SQLAusdruckRef;

ddpbool Oeffne_DB(DatenbankRef p, ddpstring *db_name) {
	struct sqlite3 **db = (struct sqlite3 **)p;
	int err = sqlite3_open(db_name->str, db);
	if (err != SQLITE_OK) {
		fprintf(stderr, "Can't open database: %s\n", sqlite3_errmsg(*db));
		sqlite3_close(*db);
		return false;
	}
	return true;
}

void Schließe_DB(Datenbank db) {
	sqlite3_close((struct sqlite3 *)db);
}

ddpbool Statement_Vorbereiten(Datenbank pDB, SQLAusdruckRef pStmt,
							  ddpstring *sql) {
	struct sqlite3 *db = (struct sqlite3 *)pDB;
	struct sqlite3_stmt **stmt = (struct sqlite3_stmt **)pStmt;
	int err = sqlite3_prepare_v2(db, sql->str, -1, stmt, NULL);
	if (err != SQLITE_OK) {
		fprintf(stderr, "Error in preparing statement: %s\n", sqlite3_errmsg(db));
		return false;
	}

	return true;
}

void Statement_Schließen(SQLAusdruck stmt) {
	sqlite3_finalize((struct sqlite3_stmt *)stmt);
}

ddpbool Naechste_Zeile_Vorbereiten(SQLAusdruck stmt) {
	return sqlite3_step((struct sqlite3_stmt *)stmt) == SQLITE_ROW;
}

void Lies_Text(ddpstring *ret, SQLAusdruck stmt) {
	const uint8_t *text = sqlite3_column_text((struct sqlite3_stmt *)stmt, 0);
	*ret = DDP_EMPTY_STRING;

	if (!text) {
		return;
	}

	int len = strlen((const char *)text);
	ret->cap = len + 1;
	ret->str = DDP_ALLOCATE(char, ret->cap);
	memcpy(ret->str, text, len);
}
