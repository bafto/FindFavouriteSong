#include "lib/sqlite3.h"
#include <stdint.h>
#include <stdio.h>

int test(void) {
	struct sqlite3 *db;
	int err = sqlite3_open("test.db", &db);
	if (err != SQLITE_OK) {
		fprintf(stderr, "Can't open database: %s\n", sqlite3_errmsg(db));
		sqlite3_close(db);
		return 1;
	}

	struct sqlite3_stmt *stmt;
	err = sqlite3_prepare_v2(db, "SELECT name FROM people", -1, &stmt, NULL);
	if (err != SQLITE_OK) {
		fprintf(stderr, "Error in preparing statement: %s", sqlite3_errmsg(db));
		sqlite3_close(db);
		return 1;
	}

	while ((err = sqlite3_step(stmt)) == SQLITE_ROW) {
		const uint8_t *text = sqlite3_column_text(stmt, 0);

		printf("Result: %s\n", text);
	}
	if (err != SQLITE_DONE) {
		fprintf(stderr, "Error in reading statement(%d): %s", err, sqlite3_errmsg(db));
		sqlite3_close(db);
		return 1;
	}

	sqlite3_finalize(stmt);
	sqlite3_close(db);
	return 0;
}

int main(int argc, char *argv[]) {
	test();
}
