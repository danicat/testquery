-- Copyright (c) 2026 Google LLC
--
-- Permission is hereby granted, free of charge, to any person obtaining a copy
-- of this software and associated documentation files (the "Software"), to deal
-- in the Software without restriction, including without limitation the rights
-- to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
-- copies of the Software, and to permit persons to whom the Software is
-- furnished to do so, subject to the following conditions:
--
-- The above copyright notice and this permission notice shall be included in
-- all copies or substantial portions of the Software.
--
-- THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
-- IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
-- FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
-- AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
-- LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
-- OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
-- SOFTWARE.

CREATE TABLE all_tests (
	"time" TIMESTAMP NOT NULL,
	"action" TEXT NOT NULL,
	status TEXT NOT NULL,
	package TEXT NOT NULL,
	test TEXT NOT NULL,
	parent_test TEXT NULL,
	is_subtest INTEGER NOT NULL DEFAULT 0,
	is_parent INTEGER NOT NULL DEFAULT 0,
	elapsed NUMERIC NULL,
	"output" TEXT NULL,
	PRIMARY KEY (package, test)
);

CREATE TABLE metadata (
	key TEXT NOT NULL PRIMARY KEY,
	value TEXT NOT NULL
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
	test_package TEXT NOT NULL DEFAULT '',
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

CREATE VIEW leaf_tests AS
SELECT *
FROM all_tests
WHERE is_parent = 0;

CREATE VIEW failed_tests AS
SELECT package, test, elapsed, "output"
FROM all_tests
WHERE action = 'fail' AND is_parent = 0;

CREATE VIEW passed_tests AS
SELECT package, test, elapsed
FROM all_tests
WHERE action = 'pass' AND is_parent = 0;

CREATE VIEW slow_tests AS
SELECT package, test, action, elapsed
FROM all_tests
WHERE is_parent = 0
ORDER BY elapsed DESC;

CREATE VIEW test_summary AS
SELECT
	package,
	COUNT(*) FILTER (WHERE is_parent = 0) AS leaf_tests,
	COUNT(*) FILTER (WHERE is_subtest = 1 AND is_parent = 0) AS subtests,
	COUNT(*) AS total_events,
	COUNT(*) FILTER (WHERE is_parent = 0 AND action = 'pass') AS passed,
	COUNT(*) FILTER (WHERE is_parent = 0 AND action = 'fail') AS failed,
	ROUND(COALESCE(SUM(elapsed) FILTER (WHERE is_parent = 0), 0), 4) AS total_elapsed
FROM all_tests
GROUP BY package;

CREATE VIEW missing_coverage AS
SELECT package, function_name, file, start_line, start_col, end_line, end_col
FROM all_coverage
WHERE count = 0;

CREATE VIEW code_coverage AS
SELECT
	ac.package,
	ac.file,
	ac.line_number,
	ac.content,
	MAX(IFNULL(cov.count, 0)) AS covered
FROM all_code ac
LEFT JOIN all_coverage cov
  ON ac.file = cov.file
 AND ac.line_number BETWEEN cov.start_line AND cov.end_line
WHERE ac.file NOT LIKE '%_test.go'
GROUP BY ac.package, ac.file, ac.line_number, ac.content;

CREATE VIEW file_coverage AS
SELECT
	package,
	file,
	SUM(CASE WHEN count > 0 THEN stmt_num ELSE 0 END) AS covered_stmts,
	SUM(stmt_num) AS total_stmts,
	ROUND(100.0 * SUM(CASE WHEN count > 0 THEN stmt_num ELSE 0 END) / NULLIF(SUM(stmt_num), 0), 2) AS coverage_pct
FROM all_coverage
GROUP BY package, file;

CREATE VIEW package_coverage AS
SELECT
	package,
	SUM(CASE WHEN count > 0 THEN stmt_num ELSE 0 END) AS covered_stmts,
	SUM(stmt_num) AS total_stmts,
	ROUND(100.0 * SUM(CASE WHEN count > 0 THEN stmt_num ELSE 0 END) / NULLIF(SUM(stmt_num), 0), 2) AS coverage_pct
FROM all_coverage
GROUP BY package;

CREATE VIEW uncovered_lines AS
SELECT DISTINCT
	ac.package,
	ac.file,
	ac.line_number,
	ac.content,
	cov.function_name
FROM all_code ac
JOIN all_coverage cov
  ON ac.file = cov.file
 AND ac.line_number BETWEEN cov.start_line AND cov.end_line
WHERE cov.count = 0
  AND ac.file NOT LIKE '%_test.go';
