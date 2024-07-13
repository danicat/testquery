	CREATE TABLE all_tests (
		"time" TIMESTAMP NOT NULL,
		"action" TEXT NOT NULL,
		package TEXT NOT NULL,
        test TEXT NOT NULL,
        elapsed NUMERIC NULL,
        "output" TEXT NULL
	);

    CREATE TABLE all_coverage (
		package TEXT NOT NULL,
		file TEXT NOT NULL,
		from_line INTEGER NOT NULL,
		from_col INTEGER NOT NULL,
		to_line INTEGER NOT NULL,
		to_col INTEGER NOT NULL,
		stmt_num INTEGER NOT NULL,
		count INTEGER NOT NULL
	);

    CREATE TABLE test_coverage (
		test_name TEXT NOT NULL,
		package TEXT NOT NULL,
		file TEXT NOT NULL,
		from_line INTEGER NOT NULL,
		from_col INTEGER NOT NULL,
		to_line INTEGER NOT NULL,
		to_col INTEGER NOT NULL,
		stmt_num INTEGER NOT NULL,
		count INTEGER NOT NULL,
		function TEXT NOT NULL
	);
