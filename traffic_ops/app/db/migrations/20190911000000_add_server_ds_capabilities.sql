/*

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
*/

-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

CREATE TABLE IF NOT EXISTS server_capability_types (
    name TEXT NOT NULL,
    last_updated timestamp with time zone NOT NULL DEFAULT NOW(),

    PRIMARY KEY (name)
);

CREATE TABLE IF NOT EXISTS server_capabilities (
    server_id int REFERENCES server (id) ON UPDATE CASCADE NOT NULL,
    capability_name TEXT REFERENCES server_capability_types (name) ON UPDATE CASCADE NOT NULL,
    last_updated timestamp with time zone NOT NULL DEFAULT NOW(),

   PRIMARY KEY (server_id, capability_name)
);

CREATE TABLE IF NOT EXISTS deliveryservice_required_capabilities (
    deliveryservice_id int REFERENCES deliveryservice (id) ON UPDATE CASCADE NOT NULL,
    capability_name TEXT REFERENCES server_capability_types (name) ON UPDATE CASCADE NOT NULL,
    last_updated timestamp with time zone NOT NULL DEFAULT NOW(),

    PRIMARY KEY (deliveryservice_id, capability_name)
);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE IF EXISTS deliveryservice_required_capabilities;
DROP TABLE IF EXISTS server_capabilities;
DROP TABLE IF EXISTS server_capability_types;

