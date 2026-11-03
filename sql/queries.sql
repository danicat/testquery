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

-- demo
select package, test, action from all_tests;
select * from passed_tests;
select * from failed_tests;
select function_name, file, start_line, end_line from missing_coverage;
select test_name, function_name, start_line, end_line count from test_coverage where count > 0;
select file, line_number, content from all_code limit 10;

select file, line_number, content from all_code where (file, line_number) in (select file, start_line from test_coverage where count = 0);

select distinct ac.file, line_number, content, ifnull(count, 0) covered from all_code ac left join all_coverage cov on ac.file = cov.file and ac.line_number between cov.start_line and cov.end_line where ac.file not like '%_test.go';