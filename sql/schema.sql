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
		start_line INTEGER NOT NULL,
		start_col INTEGER NOT NULL,
		end_line INTEGER NOT NULL,
		end_col INTEGER NOT NULL,
		stmt_num INTEGER NOT NULL,
		count INTEGER NOT NULL,
		function_name TEXT NOT NULL
	);

    CREATE TABLE test_coverage (
		test_name TEXT NOT NULL,
		package TEXT NOT NULL,
		file TEXT NOT NULL,
		start_line INTEGER NOT NULL,
		start_col INTEGER NOT NULL,
		end_line INTEGER NOT NULL,
		end_col INTEGER NOT NULL,
		stmt_num INTEGER NOT NULL,
		count INTEGER NOT NULL,
		function_name TEXT NULL
	);

	CREATE TABLE all_code (
		package TEXT NOT NULL,
		file TEXT NOT NULL,
		line_number INTEGER NOT NULL,
		content TEXT NOT NULL
	);

create view failed_tests as
select package, test
  from all_tests
 where action = 'fail';

create view passed_tests as 
select package, test
  from all_tests
 where action = 'pass';

create view missing_coverage as
select package, function_name, file, start_line, start_col, end_line, end_col
  from all_coverage
 where count = 0;