-- Copyright 2020 Red Hat, Inc
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--      http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

insert into report (org_id, cluster, report, reported_at, last_checked_at) values (1, '00000000-0000-0000-0000-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
insert into report (org_id, cluster, report, reported_at, last_checked_at) values (1, '00000000-0000-0000-ffff-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
insert into report (org_id, cluster, report, reported_at, last_checked_at) values (1, '00000000-0000-0000-0000-ffffffffffff', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
insert into report (org_id, cluster, report, reported_at, last_checked_at) values (2, '00000000-ffff-0000-0000-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
insert into report (org_id, cluster, report, reported_at, last_checked_at) values (2, '00000000-0000-ffff-0000-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
insert into report (org_id, cluster, report, reported_at, last_checked_at) values (3, 'aaaaaaaa-0000-0000-0000-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
insert into report (org_id, cluster, report, reported_at, last_checked_at) values (3, 'addddddd-0000-0000-0000-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
insert into report (org_id, cluster, report, reported_at, last_checked_at) values (4, 'addddddd-bbbb-0000-0000-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
insert into report (org_id, cluster, report, reported_at, last_checked_at) values (4, 'addddddd-bbbb-cccc-0000-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
