-- demo
select package, test, action from all_tests;
select * from passed_tests;
select * from failed_tests;
select function_name, file, start_line, end_line from missing_coverage;
select test_name, function_name, start_line, end_line count from test_coverage where count > 0;

select distinct function_name from test_coverage where test_name in (select test from all_tests where action = 'fail');
