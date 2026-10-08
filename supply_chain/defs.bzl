"""Public supply-chain policy and notice rules."""

load("//supply_chain/private:advisory_index.bzl", _advisory_index = "advisory_index")
load("//supply_chain/private:supply_chain_test.bzl", _supply_chain_test = "supply_chain_test")
load("//supply_chain/private:third_party_notices.bzl", _third_party_notices = "third_party_notices")

visibility("public")
advisory_index = _advisory_index
supply_chain_test = _supply_chain_test
third_party_notices = _third_party_notices
