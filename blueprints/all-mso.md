<!--
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
-->
# Simplify Multi-Site Origin and Remove Single-Origin Configuration

## Problem Description
MSO is a pain to configure and use, but offers some distinct advantages, and no downsides, over non-MSO:
- The Cache Key is easy to preserve or change, without custom cache key plugin pains (because it's a static URL string, unrelated to the actual origins).
- Even if a customer only has 1 Origin today, it's easy to add more origins in the future.

## Proposed Change

MSO configuration is always used. Non-MSO no longer exists. MSO is fixed to be easy to configure.

1. Traffic Portal UI Delivery Services:
    - A list box of Origins
    - A radio button for request behavior: (a) round-robin (b) consistent-hash (c) ordered failover
    - A list box of HTTP codes to consider an origin failure (default: 500,502,503) and retry rather than returning to the client
    - Field "Origin Server Base URL" is renamed "Cache Key Base URL" or similar (which is how it behaves for MSO)
        - readonly and auto-populated with the first origin on DS creation
        - still modifiable when editing the DS later

2. ORT Config Gen:
    - Use Origins table instead of existing MSO Servers table
    - Use DS request behavior and DS HTTP Failure Codes instead of Server Profile Parameters, for MSO behavior

3. Traffic Ops Upgrade Migration:
    - Database Migration, if possible
    - Convert non-MSO to MSO. For Non-MSO DSes, create Origin table entry
    - Convert MSO Servers to Origins, set Origins in DS List box (new 1:* table?)
    - Set Delivery Services Request Behavior and Failure Codes from DS Profile Parameters


### Traffic Portal Impact
Delivery Services UI page is changed, as above.

### Traffic Ops Impact
See REST API Impact

#### REST API Impact
Delivery Services API is changed, as above. Delivery Services are given new Origin List, Request Behavior, and HTTP Failure Codes fields.

#### Client Impact
None.

#### Data Model / Database Impact
TO Database adds new DS fields, new 1:* table for DS Origins list, new 1:* table for DS Failure Codes list.
Migration to convert old non-MSO and MSO to new fields.

### ORT Impact
ORT is changed as above, to use Origins instead of Servers, consider new DS Request Behavior and Failure Codes, delete old non-MSO code.

### Traffic Monitor Impact
None.

### Traffic Router Impact
None.

### Traffic Stats Impact
None.

### Traffic Vault Impact
None.

### Documentation Impact
Docs for new DS fields. Delete MSO Docs. No specific "MSO Docs."

Delivery Services have as many origins as you like, simple, no more explanation is necessary.

### Testing Impact

TO API Tests for new DS fields. ORT unit tests for MSO changes.

### Performance Impact
None expected. TO changes should not significantly increase API size or cost. ATS MSO configuration is almost identical to non-MSO, simply a list instead of a single parent in the same field, should not change performance.

It should be a goal for ORT to take less than 15 seconds to run, given a fast Traffic Ops. Because a faster ORT means TC changes propogate faster, which is ideal. But ORT is not in the Request Path, so performance is not critical to TC operation. Further, the previous release of ORT took 5-8 minutes on a large CDN.

### Security Impact
None.

### Upgrade Impact
None. Upgrade should automatically convert existing MSO/Single DSes to new fields.

### Operations Impact
Operators should be told how Origins now work, but it should be easy to understand.
New configuration is drastically simpler than old MSO, and a very small change from old non-MSO.

### Developer Impact
Development should be easier, much less complexity in TO and TP, much less code in ORT than separate MSO/Single paths.

## Alternatives
Keep MSO the same
- No advantages, unnecessarily complex, confusing, and painful.

Simplify MSO, but still keep "single-origin DS"
- Unnecessarily complex and confusing. Still requires additional MSO DS fields, with additional non-MSO fields
- Preserving Cache with origin URL changes is difficult and easy to break
- Converting to MSO when a tenant needs an additional origin is painful and easy to break

## Dependencies
None.

## References
https://traffic-control-cdn.readthedocs.io/en/latest/admin/quick_howto/multi_site.html?highlight=mso
https://traffic-control-cdn.readthedocs.io/en/latest/overview/delivery_services.html?highlight=mso#use-multi-site-origin-feature
