#!/usr/bin/env bash
# E2E smoke test for the vertical slice. Assumes `docker compose up -d --build`
# is already running. Drives real HTTP through gl/ap, then verifies audit consumed
# the emitted Kafka events. Exits non-zero if any check fails.
set -uo pipefail

GL=http://localhost:8081
AP=http://localhost:8082
AR=http://localhost:8084
COA=http://localhost:8085
TAX=http://localhost:8086
TRS=http://localhost:8087
USR=http://localhost:8088
SEC=http://localhost:8089
MD=http://localhost:8090
ORG=http://localhost:8091
WF=http://localhost:8092
PROC=http://localhost:8093
INV=http://localhost:8094
MFG=http://localhost:8095
SO=http://localhost:8096
HR=http://localhost:8097
PAY=http://localhost:8098
REP=http://localhost:8099
CUR=http://localhost:8100
FA=http://localhost:8101
DIM=http://localhost:8102
BDG=http://localhost:8103
FCL=http://localhost:8104
CON=http://localhost:8105
PRJ=http://localhost:8106
AST=http://localhost:8107
MNT=http://localhost:8108
CRM=http://localhost:8109
LOC=http://localhost:8110
AUD=http://localhost:8083
GW=http://localhost:8000/api/v1

ORGA=11111111-1111-1111-1111-111111111111
ORGB=22222222-2222-2222-2222-222222222222
USER=33333333-3333-3333-3333-333333333333
FINORG=10000000-0000-0000-0000-000000000001
LEDGER=10000000-0000-0000-0000-000000000002
PERIOD=10000000-0000-0000-0000-000000000003
ACC1=10000000-0000-0000-0000-00000000000a
ACC2=10000000-0000-0000-0000-00000000000b
VENDOR=10000000-0000-0000-0000-00000000000c
CUSTOMER=10000000-0000-0000-0000-00000000000d
TS=$(date +%H%M%S)

pass=0; fail=0
ck() { # ck "desc" actual expected
  if [ "$2" = "$3" ]; then echo "  PASS: $1 ($2)"; pass=$((pass+1));
  else echo "  FAIL: $1 (got '$2', want '$3')"; fail=$((fail+1)); fi
}
ckc() { # ck contains: "desc" haystack needle
  if printf '%s' "$2" | grep -q "$3"; then echo "  PASS: $1"; pass=$((pass+1));
  else echo "  FAIL: $1 (missing '$3')"; fail=$((fail+1)); fi
}

echo "== waiting for services =="
for p in 8081 8082 8083 8084 8085 8086 8087 8088 8089 8090 8091 8092 8093 8094 8095 8096 8097 8098 8099 8100 8101 8102 8103 8104 8105 8106 8107 8108 8109 8110 8000; do
  for i in $(seq 1 60); do curl -fsS "localhost:$p/readyz" >/dev/null 2>&1 && break; sleep 2; done
done

AH=(-H "X-Organization-Id: $ORGA" -H "X-User-Id: $USER" -H "X-Roles: SYSTEM_ADMIN" -H "Content-Type: application/json")

echo "== 1. gl: create balanced journal =="
JBODY='{"journalNo":"JV-'"$TS"'","financialOrganizationId":"'"$FINORG"'","ledgerId":"'"$LEDGER"'","fiscalPeriodId":"'"$PERIOD"'","postingDate":"2026-03-15","currencyCode":"IDR","lines":[{"lineNo":1,"accountId":"'"$ACC1"'","debitAmount":"1000.0000","creditAmount":"0","functionalAmount":"1000.0000"},{"lineNo":2,"accountId":"'"$ACC2"'","debitAmount":"0","creditAmount":"1000.0000","functionalAmount":"1000.0000"}]}'
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: k-$TS" -d "$JBODY" "$GL/journal-entries")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create journal -> 201" "$CODE" "201"
JID=$(printf '%s' "$BODY" | jq -r .id); JVER=$(printf '%s' "$BODY" | jq -r .rowVersion)

echo "== 1b. gl: idempotent replay (same key -> same id) =="
JID2=$(curl -sS "${AH[@]}" -H "Idempotency-Key: k-$TS" -d "$JBODY" "$GL/journal-entries" | jq -r .id)
ck "idempotent replay id" "$JID2" "$JID"

echo "== 2. gl: post journal (emits gl.journal.posted) =="
CODE=$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -H "If-Match: $JVER" -X POST "$GL/journal-entries/$JID/post")
ck "post journal -> 200" "$CODE" "200"

echo "== 2b. gl: stale post -> 409 (optimistic lock) =="
CODE=$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -H "If-Match: $JVER" -X POST "$GL/journal-entries/$JID/post")
ck "stale post -> 409" "$CODE" "409"

echo "== 3. gl: RLS — org B cannot see org A entry -> 404 =="
CODE=$(curl -sS -o /dev/null -w '%{http_code}' -H "X-Organization-Id: $ORGB" "$GL/journal-entries/$JID")
ck "RLS org B get -> 404" "$CODE" "404"

echo "== 3b. gl: deny-by-default — no scope -> 401 =="
CODE=$(curl -sS -o /dev/null -w '%{http_code}' "$GL/journal-entries")
ck "no scope list -> 401" "$CODE" "401"

echo "== 4. gl: unbalanced journal post -> 422 (BR-556) =="
UBODY='{"journalNo":"JVU-'"$TS"'","financialOrganizationId":"'"$FINORG"'","ledgerId":"'"$LEDGER"'","fiscalPeriodId":"'"$PERIOD"'","postingDate":"2026-03-15","currencyCode":"IDR","lines":[{"lineNo":1,"accountId":"'"$ACC1"'","debitAmount":"1000.0000","creditAmount":"0","functionalAmount":"1000.0000"},{"lineNo":2,"accountId":"'"$ACC2"'","debitAmount":"0","creditAmount":"500.0000","functionalAmount":"500.0000"}]}'
UJID=$(curl -sS "${AH[@]}" -H "Idempotency-Key: ku-$TS" -d "$UBODY" "$GL/journal-entries" | jq -r .id)
UVER=1
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "If-Match: $UVER" -X POST "$GL/journal-entries/$UJID/post")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "unbalanced post -> 422" "$CODE" "422"
ckc "422 code = GL_JOURNAL_UNBALANCED" "$BODY" "GL_JOURNAL_UNBALANCED"

echo "== 5. ap: create + record invoice (emits ap.invoice.recorded) =="
IBODY='{"invoiceNo":"INV-'"$TS"'","vendorId":"'"$VENDOR"'","invoiceDate":"2026-03-15","dueDate":"2026-04-15","currencyCode":"IDR","totalAmount":"500.0000","lines":[{"lineNo":1,"accountId":"'"$ACC1"'","amount":"500.0000","description":"jasa"}]}'
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: i-$TS" -d "$IBODY" "$AP/vendor-invoices")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create invoice -> 201" "$CODE" "201"
IID=$(printf '%s' "$BODY" | jq -r .id); IVER=$(printf '%s' "$BODY" | jq -r .rowVersion)
CODE=$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -H "If-Match: $IVER" -X POST "$AP/vendor-invoices/$IID/record")
ck "record invoice -> 200" "$CODE" "200"

echo "== 5b. ar: create customer invoice (emits ar.invoice.issued on create) =="
CBODY='{"invoiceNo":"AR-'"$TS"'","customerId":"'"$CUSTOMER"'","invoiceDate":"2026-03-15","dueDate":"2026-04-15","currencyCode":"IDR","totalAmount":"750.0000","lines":[{"lineNo":1,"accountId":"'"$ACC1"'","amount":"750.0000","description":"penjualan"}]}'
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: ar-$TS" -d "$CBODY" "$AR/customer-invoices")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create customer invoice -> 201" "$CODE" "201"
ARID=$(printf '%s' "$BODY" | jq -r .id)
ck "ar invoice status Issued" "$(printf '%s' "$BODY" | jq -r .status)" "Issued"

echo "== 5c. coa: create account (emits coa.account.upserted) =="
CHART=$(curl -sS "${AH[@]}" -d '{"code":"COA-'"$TS"'","name":"Main COA"}' "$COA/charts-of-accounts" | jq -r .id)
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: acc-$TS" -d '{"chartOfAccountsId":"'"$CHART"'","code":"1000-'"$TS"'","name":"Kas","accountType":"Asset","allowPosting":true}' "$COA/accounts")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create account -> 201" "$CODE" "201"
ACCID=$(printf '%s' "$BODY" | jq -r .id)

echo "== 5d. tax: create tax invoice (emits tax.invoice.issued) =="
TAXCODE=$(curl -sS "${AH[@]}" -d '{"code":"PPN-'"$TS"'","rate":"11.0000","taxType":"VAT","glAccountId":"'"$ACC1"'"}' "$TAX/tax-codes" | jq -r .id)
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: tax-$TS" -d '{"sourceDocumentRef":"SRC-'"$TS"'","taxCodeId":"'"$TAXCODE"'","baseAmount":"1000.0000","taxAmount":"110.0000","efakturNo":"EF-'"$TS"'"}' "$TAX/tax-invoices")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create tax invoice -> 201" "$CODE" "201"
TAXID=$(printf '%s' "$BODY" | jq -r .id)

echo "== 5e. treasury: create + settle payment (emits treasury.payment.settled) =="
BANK=$(curl -sS "${AH[@]}" -d '{"code":"BANK-'"$TS"'","bankName":"BCA","accountNo":"123","currencyCode":"IDR","glAccountId":"'"$ACC1"'"}' "$TRS/bank-accounts" | jq -r .id)
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: pay-$TS" -d '{"paymentNo":"PAY-'"$TS"'","paymentType":"Outgoing","bankAccountId":"'"$BANK"'","amount":"500.0000","paymentDate":"2026-03-15"}' "$TRS/payments")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create payment -> 201" "$CODE" "201"
PAYID=$(printf '%s' "$BODY" | jq -r .id); PAYVER=$(printf '%s' "$BODY" | jq -r .rowVersion)
CODE=$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -H "If-Match: $PAYVER" -X POST "$TRS/payments/$PAYID/settle")
ck "settle payment -> 200" "$CODE" "200"

echo "== 5f. user: create app user (emits user.account.upserted) =="
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: usr-$TS" -d '{"username":"u'"$TS"'","email":"u'"$TS"'@ex.co","fullName":"User '"$TS"'"}' "$USR/users")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create user -> 201" "$CODE" "201"
USRID=$(printf '%s' "$BODY" | jq -r .id)

echo "== 5g. security: create role (emits security.role.upserted) =="
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: role-$TS" -d '{"code":"ROLE-'"$TS"'","name":"Officer","isSystem":false}' "$SEC/roles")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create role -> 201" "$CODE" "201"
ROLEID=$(printf '%s' "$BODY" | jq -r .id)

echo "== 5h. masterdata: create party (emits masterdata.party.upserted) =="
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: party-$TS" -d '{"code":"CUST-'"$TS"'","name":"Cust '"$TS"'","partyType":"customer"}' "$MD/parties")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create party -> 201" "$CODE" "201"
PARTYID=$(printf '%s' "$BODY" | jq -r .id)

echo "== 5i. org: create organization (emits org.unit.upserted) =="
OTYPE=$(curl -sS "${AH[@]}" -d '{"code":"TYPE-'"$TS"'","name":"Company"}' "$ORG/organization-types" | jq -r .id)
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: org-$TS" -d '{"code":"ORG-'"$TS"'","name":"Unit '"$TS"'","organizationTypeId":"'"$OTYPE"'","effectiveFrom":"2026-01-01"}' "$ORG/organizations")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create organization -> 201" "$CODE" "201"
ORGUNITID=$(printf '%s' "$BODY" | jq -r .id)

echo "== 5j. workflow: create instance (emits workflow.approval.requested) =="
WFDEF=$(curl -sS "${AH[@]}" -d '{"code":"WF-'"$TS"'","documentType":"PurchaseOrder"}' "$WF/workflow-definitions" | jq -r .id)
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: wf-$TS" -d '{"definitionId":"'"$WFDEF"'","documentRef":"PO-'"$TS"'"}' "$WF/workflow-instances")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create workflow instance -> 201" "$CODE" "201"
WFID=$(printf '%s' "$BODY" | jq -r .id)

echo "== 5k. procurement: create + approve PO (emits procurement.po.approved) =="
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: po-$TS" -d '{"poNo":"PO-'"$TS"'","vendorId":"'"$VENDOR"'","orderDate":"2026-03-15","currencyCode":"IDR","totalAmount":"900.0000","lines":[{"lineNo":1,"itemId":"'"$ACC1"'","orderQty":"3","unitPrice":"300.0000"}]}' "$PROC/purchase-orders")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create PO -> 201" "$CODE" "201"
POID=$(printf '%s' "$BODY" | jq -r .id); POVER=$(printf '%s' "$BODY" | jq -r .rowVersion)
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "If-Match: $POVER" -X POST "$PROC/purchase-orders/$POID/approve")
CODE=$(printf '%s' "$R" | tail -1); POVER2=$(printf '%s' "$R" | sed '$d' | jq -r .rowVersion)
ck "approve PO -> 200" "$CODE" "200"
CODE=$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -H "If-Match: $POVER2" -X POST "$PROC/purchase-orders/$POID/receive")
ck "receive PO (emits procurement.goods.received) -> 200" "$CODE" "200"

echo "== 5l. inventory: seed stock + create & issue reservation (emits inventory.stock.reserved) =="
WH=$(curl -sS "${AH[@]}" -d '{"code":"WH-'"$TS"'","name":"Gudang","organizationRefId":"'"$ORGA"'"}' "$INV/warehouses" | jq -r .id)
CODE=$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -H "Idempotency-Key: stk-$TS" -d '{"warehouseId":"'"$WH"'","itemId":"'"$ACC1"'","movementType":"Receipt","quantity":"100.0000","unitCost":"300.0000","sourceDocumentRef":"INIT-'"$TS"'"}' "$INV/stock-movements")
ck "seed inventory stock movement -> 201" "$CODE" "201"
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: rsv-$TS" -d '{"itemId":"'"$ACC1"'","warehouseId":"'"$WH"'","reservedQty":"5"}' "$INV/reservations")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create reservation -> 201" "$CODE" "201"
RESID=$(printf '%s' "$BODY" | jq -r .id); RESVER=$(printf '%s' "$BODY" | jq -r .rowVersion)
CODE=$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -H "If-Match: $RESVER" -d '{"amount":"640.0000"}' "$INV/reservations/$RESID/issue")
ck "issue reservation (emits inventory.goods.issued) -> 200" "$CODE" "200"

echo "== 5m. manufacturing: create + release + complete work order (emits manufacturing.workorder.completed) =="
BOM=$(curl -sS "${AH[@]}" -d '{"code":"BOM-'"$TS"'","productItemId":"'"$ACC1"'"}' "$MFG/bill-of-materials" | jq -r .id)
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: wo-$TS" -d '{"woNo":"WO-'"$TS"'","productItemId":"'"$ACC1"'","bomId":"'"$BOM"'","plannedQty":"10"}' "$MFG/work-orders")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create work order -> 201" "$CODE" "201"
WOID=$(printf '%s' "$BODY" | jq -r .id); WOVER=$(printf '%s' "$BODY" | jq -r .rowVersion)
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "If-Match: $WOVER" -X POST "$MFG/work-orders/$WOID/release")
CODE=$(printf '%s' "$R" | tail -1); WOVER2=$(printf '%s' "$R" | sed '$d' | jq -r .rowVersion)
ck "release work order -> 200" "$CODE" "200"
CODE=$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -H "If-Match: $WOVER2" -X POST "$MFG/work-orders/$WOID/complete")
ck "complete work order -> 200" "$CODE" "200"

echo "== 5n. sales: create + confirm SO (emits sales.order.confirmed) =="
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: so-$TS" -d '{"soNo":"SO-'"$TS"'","customerId":"'"$CUSTOMER"'","orderDate":"2026-03-15","currencyCode":"IDR","totalAmount":"1200.0000","lines":[{"lineNo":1,"itemId":"'"$ACC1"'","orderQty":"4","unitPrice":"300.0000"}]}' "$SO/sales-orders")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create SO -> 201" "$CODE" "201"
SOID=$(printf '%s' "$BODY" | jq -r .id); SOVER=$(printf '%s' "$BODY" | jq -r .rowVersion)
CODE=$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -H "If-Match: $SOVER" -X POST "$SO/sales-orders/$SOID/confirm")
ck "confirm SO -> 200" "$CODE" "200"

echo "== 5o. hr: create employee (emits hr.employee.updated) =="
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: emp-$TS" -d '{"employeeNo":"EMP-'"$TS"'","fullName":"Budi","hireDate":"2026-01-05"}' "$HR/employees")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create employee -> 201" "$CODE" "201"
EMPID=$(printf '%s' "$BODY" | jq -r .id)

echo "== 5p. payroll: create + calculate + approve + post run (emits payroll.run.calculated -> approved -> posted/completed) =="
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: run-$TS" -d '{"runNo":"RUN-'"$TS"'","periodMonth":3,"periodYear":2026}' "$PAY/payroll-runs")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create payroll run -> 201" "$CODE" "201"
RUNID=$(printf '%s' "$BODY" | jq -r .id); RUNVER=$(printf '%s' "$BODY" | jq -r .rowVersion)
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "If-Match: $RUNVER" -d '{"employees":[{"employeeId":"'"$EMPID"'","basicSalary":"5000000.0000","ptkp":"TK/0","hasNpwp":true}]}' "$PAY/payroll-runs/$RUNID/calculate")
CODE=$(printf '%s' "$R" | tail -1); RUNVER2=$(printf '%s' "$R" | sed '$d' | jq -r .rowVersion)
ck "calculate payroll run -> 200" "$CODE" "200"
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "If-Match: $RUNVER2" -X POST "$PAY/payroll-runs/$RUNID/approve")
CODE=$(printf '%s' "$R" | tail -1); RUNVER3=$(printf '%s' "$R" | sed '$d' | jq -r .rowVersion)
ck "approve payroll run -> 200" "$CODE" "200"
CODE=$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -H "If-Match: $RUNVER3" -d '{"glJournalRef":"AUTO-RUN-'"$TS"'"}' "$PAY/payroll-runs/$RUNID/post")
ck "post payroll run (emits payroll.run.completed) -> 200" "$CODE" "200"

echo "== 6. audit + reporting: poll until all events consumed/projected (<=40s) =="
found_gl=0; found_ap=0; found_ar=0; found_coa=0; found_tax=0; found_trs=0; found_usr=0; found_sec=0; found_md=0; found_org=0; found_wf=0; found_proc=0; found_inv=0; found_mfg=0; found_so=0; found_hr=0; found_pay=0; found_rep=0; no_double_count=1
for i in $(seq 1 20); do
  A=$(curl -sS -H "X-Organization-Id: $ORGA" "$AUD/audit-events?size=500")
  printf '%s' "$A" | jq -e '.data[] | select(.action=="gl.journal.posted" and .entityId=="'"$JID"'")' >/dev/null 2>&1 && found_gl=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="ap.invoice.recorded" and .entityId=="'"$IID"'")' >/dev/null 2>&1 && found_ap=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="ar.invoice.issued" and .entityId=="'"$ARID"'")' >/dev/null 2>&1 && found_ar=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="coa.account.upserted" and .entityId=="'"$ACCID"'")' >/dev/null 2>&1 && found_coa=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="tax.invoice.issued" and .entityId=="'"$TAXID"'")' >/dev/null 2>&1 && found_tax=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="treasury.payment.settled" and .entityId=="'"$PAYID"'")' >/dev/null 2>&1 && found_trs=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="user.account.upserted" and .entityId=="'"$USRID"'")' >/dev/null 2>&1 && found_usr=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="security.role.upserted" and .entityId=="'"$ROLEID"'")' >/dev/null 2>&1 && found_sec=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="masterdata.party.upserted" and .entityId=="'"$PARTYID"'")' >/dev/null 2>&1 && found_md=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="org.unit.upserted" and .entityId=="'"$ORGUNITID"'")' >/dev/null 2>&1 && found_org=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="workflow.approval.requested" and .entityId=="'"$WFID"'")' >/dev/null 2>&1 && found_wf=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="procurement.po.approved" and .entityId=="'"$POID"'")' >/dev/null 2>&1 && found_proc=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="inventory.stock.reserved" and .entityId=="'"$RESID"'")' >/dev/null 2>&1 && found_inv=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="manufacturing.workorder.completed" and .entityId=="'"$WOID"'")' >/dev/null 2>&1 && found_mfg=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="sales.order.confirmed" and .entityId=="'"$SOID"'")' >/dev/null 2>&1 && found_so=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="hr.employee.updated" and .entityId=="'"$EMPID"'")' >/dev/null 2>&1 && found_hr=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="payroll.run.completed" and .entityId=="'"$RUNID"'")' >/dev/null 2>&1 && found_pay=1
  RS=$(curl -sS -H "X-Organization-Id: $ORGA" "$REP/gl-balance-snapshots?size=200")
  printf '%s' "$RS" | jq -e '.data[] | select(.accountCode=="ACCT:'"$ACC1"'")' >/dev/null 2>&1 && found_rep=1
  printf '%s' "$RS" | jq -e '.data[] | select(.accountCode=="JV-'"$TS"'")' >/dev/null 2>&1 && no_double_count=0
  [ "$found_gl" = 1 ] && [ "$found_ap" = 1 ] && [ "$found_ar" = 1 ] && [ "$found_coa" = 1 ] && [ "$found_tax" = 1 ] && [ "$found_trs" = 1 ] && [ "$found_usr" = 1 ] && [ "$found_sec" = 1 ] && [ "$found_md" = 1 ] && [ "$found_org" = 1 ] && [ "$found_wf" = 1 ] && [ "$found_proc" = 1 ] && [ "$found_inv" = 1 ] && [ "$found_mfg" = 1 ] && [ "$found_so" = 1 ] && [ "$found_hr" = 1 ] && [ "$found_pay" = 1 ] && [ "$found_rep" = 1 ] && break
  sleep 2
done
ck "audit consumed gl.journal.posted" "$found_gl" "1"
ck "audit consumed ap.invoice.recorded" "$found_ap" "1"
ck "audit consumed ar.invoice.issued" "$found_ar" "1"
ck "audit consumed coa.account.upserted" "$found_coa" "1"
ck "audit consumed tax.invoice.issued" "$found_tax" "1"
ck "audit consumed treasury.payment.settled" "$found_trs" "1"
ck "audit consumed user.account.upserted" "$found_usr" "1"
ck "audit consumed security.role.upserted" "$found_sec" "1"
ck "audit consumed masterdata.party.upserted" "$found_md" "1"
ck "audit consumed org.unit.upserted" "$found_org" "1"
ck "audit consumed workflow.approval.requested" "$found_wf" "1"
ck "audit consumed procurement.po.approved" "$found_proc" "1"
ck "audit consumed inventory.stock.reserved" "$found_inv" "1"
ck "audit consumed manufacturing.workorder.completed" "$found_mfg" "1"
ck "audit consumed sales.order.confirmed" "$found_so" "1"
ck "audit consumed hr.employee.updated" "$found_hr" "1"
ck "audit consumed payroll.run.completed" "$found_pay" "1"
ck "reporting projected gl.journal.posted per account (CQRS)" "$found_rep" "1"
ck "reporting avoided journal-number double counting" "$no_double_count" "1"

echo "== 6b. audit: RLS — org B sees none of org A's audit events =="
BCOUNT=$(curl -sS -H "X-Organization-Id: $ORGB" "$AUD/audit-events?size=100" | jq '[.data[] | select(.entityId=="'"$JID"'" or .entityId=="'"$IID"'")] | length')
ck "audit RLS org B count" "$BCOUNT" "0"

echo "== 7. gl consumer: subledger events -> auto-posted GL journals (SAD 5.4.1 muara->GL) =="
# ar.invoice.issued (5b) and ap.invoice.recorded (5) are consumed by gl-service,
# which auto-posts a balanced journal (AUTO-<invoiceNo>) and emits gl.journal.posted,
# which reporting then projects per COA accountCode.
AR_AUTO="AUTO-AR-$TS"; AP_AUTO="AUTO-INV-$TS"; PAY_AUTO="AUTO-RUN-$TS"; GI_AUTO="AUTO-$RESID"
gl_ar=0; gl_ap=0; gl_pay=0; gl_gi=0; rep_ar=0; rep_ap=0; rep_pay=0; rep_gi=0
for i in $(seq 1 25); do
  J=$(curl -sS "${AH[@]}" "$GL/journal-entries?size=100")
  printf '%s' "$J" | jq -e '.data[] | select(.journalNo=="'"$AR_AUTO"'" and .status=="Posted")' >/dev/null 2>&1 && gl_ar=1
  printf '%s' "$J" | jq -e '.data[] | select(.journalNo=="'"$AP_AUTO"'" and .status=="Posted")' >/dev/null 2>&1 && gl_ap=1
  printf '%s' "$J" | jq -e '.data[] | select(.journalNo=="'"$PAY_AUTO"'" and .status=="Posted")' >/dev/null 2>&1 && gl_pay=1
  printf '%s' "$J" | jq -e '.data[] | select(.journalNo=="'"$GI_AUTO"'" and .status=="Posted")' >/dev/null 2>&1 && gl_gi=1
  S=$(curl -sS -H "X-Organization-Id: $ORGA" "$REP/gl-balance-snapshots?size=200")
  printf '%s' "$S" | jq -e '.data[] | select(.accountCode=="1120")' >/dev/null 2>&1 && rep_ar=1
  printf '%s' "$S" | jq -e '.data[] | select(.accountCode=="2110")' >/dev/null 2>&1 && rep_ap=1
  printf '%s' "$S" | jq -e '.data[] | select(.accountCode=="5300")' >/dev/null 2>&1 && rep_pay=1
  printf '%s' "$S" | jq -e '.data[] | select(.accountCode=="5200")' >/dev/null 2>&1 && rep_gi=1
  [ "$gl_ar" = 1 ] && [ "$gl_ap" = 1 ] && [ "$gl_pay" = 1 ] && [ "$gl_gi" = 1 ] && [ "$rep_ar" = 1 ] && [ "$rep_ap" = 1 ] && [ "$rep_pay" = 1 ] && [ "$rep_gi" = 1 ] && break
  sleep 2
done
ck "gl auto-posted journal from ar.invoice.issued" "$gl_ar" "1"
ck "gl auto-posted journal from ap.invoice.recorded" "$gl_ap" "1"
ck "gl auto-posted journal from payroll.run.completed (INT-003)" "$gl_pay" "1"
ck "gl auto-posted journal from inventory.goods.issued (INT-001)" "$gl_gi" "1"
ck "reporting projected AR control account 1120 (ar->gl->reporting)" "$rep_ar" "1"
ck "reporting projected AP control account 2110 (ap->gl->reporting)" "$rep_ap" "1"
ck "reporting projected payroll expense account 5300 (payroll->gl->reporting)" "$rep_pay" "1"
ck "reporting projected COGS account 5200 (inventory->gl->reporting)" "$rep_gi" "1"

echo "== 7b. INT-001 full Saga chain: sales.order.confirmed -> inventory -> ar -> gl -> reporting =="
# Confirming the SO (5n) cascades: inventory consumes it -> emits goods.issued ->
# ar consumes that -> issues invoice SI-SO-$TS -> emits ar.invoice.issued ->
# gl consumes that -> auto-posts AUTO-SI-SO-$TS -> reporting projects it.
SI_NO="SI-SO-$TS"; SI_AUTO="AUTO-SI-SO-$TS"
ar_si=0; gl_si=0; rep_si=0
for i in $(seq 1 30); do
  printf '%s' "$(curl -sS "${AH[@]}" "$AR/customer-invoices?size=200")" | jq -e '.data[] | select(.invoiceNo=="'"$SI_NO"'")' >/dev/null 2>&1 && ar_si=1
  printf '%s' "$(curl -sS "${AH[@]}" "$GL/journal-entries?size=100")" | jq -e '.data[] | select(.journalNo=="'"$SI_AUTO"'" and .status=="Posted")' >/dev/null 2>&1 && gl_si=1
  printf '%s' "$(curl -sS -H "X-Organization-Id: $ORGA" "$REP/gl-balance-snapshots?size=200")" | jq -e '.data[] | select(.accountCode=="4100")' >/dev/null 2>&1 && rep_si=1
  [ "$ar_si" = 1 ] && [ "$gl_si" = 1 ] && [ "$rep_si" = 1 ] && break
  sleep 2
done
ck "ar issued invoice from goods.issued (sales->inventory->ar)" "$ar_si" "1"
ck "gl auto-posted from derived ar invoice (ar->gl)" "$gl_si" "1"
ck "reporting projected full-chain revenue account 4100 (->reporting)" "$rep_si" "1"

echo "== 8. INT-002 chain: procurement.goods.received -> ap -> gl -> reporting =="
# Receiving the PO (5k) emits goods.received -> ap records invoice PI-PO-$TS ->
# ap.invoice.recorded -> gl auto-posts AUTO-PI-PO-$TS -> reporting projects it.
PI_NO="PI-PO-$TS"; PI_AUTO="AUTO-PI-PO-$TS"
ap_pi=0; gl_pi=0; rep_pi=0
for i in $(seq 1 30); do
  printf '%s' "$(curl -sS "${AH[@]}" "$AP/vendor-invoices?size=200")" | jq -e '.data[] | select(.invoiceNo=="'"$PI_NO"'")' >/dev/null 2>&1 && ap_pi=1
  printf '%s' "$(curl -sS "${AH[@]}" "$GL/journal-entries?size=100")" | jq -e '.data[] | select(.journalNo=="'"$PI_AUTO"'" and .status=="Posted")' >/dev/null 2>&1 && gl_pi=1
  printf '%s' "$(curl -sS -H "X-Organization-Id: $ORGA" "$REP/gl-balance-snapshots?size=200")" | jq -e '.data[] | select(.accountCode=="5100")' >/dev/null 2>&1 && rep_pi=1
  [ "$ap_pi" = 1 ] && [ "$gl_pi" = 1 ] && [ "$rep_pi" = 1 ] && break
  sleep 2
done
ck "ap recorded invoice from goods.received (procurement->ap)" "$ap_pi" "1"
ck "gl auto-posted from goods-received ap invoice (ap->gl)" "$gl_pi" "1"
ck "reporting projected INT-002 expense account 5100 (->reporting)" "$rep_pi" "1"

echo "== 9. INT-001 compensation: out-of-stock SO -> reservation.failed -> SO Cancelled =="
# An order whose requested quantity exceeds available inventory stock triggers
# inventory.reservation.failed, which sales consumes to cancel the SO (SAD 5.7).
OOSBODY='{"soNo":"OOS-'"$TS"'","customerId":"'"$CUSTOMER"'","orderDate":"2026-03-15","currencyCode":"IDR","totalAmount":"5000000.0000","lines":[{"lineNo":1,"itemId":"'"$ACC1"'","orderQty":"5000","unitPrice":"1000.0000"}]}'
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: oos-$TS" -d "$OOSBODY" "$SO/sales-orders")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "create OOS sales order -> 201" "$CODE" "201"
OOSID=$(printf '%s' "$BODY" | jq -r .id); OOSVER=$(printf '%s' "$BODY" | jq -r .rowVersion)
CODE=$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -H "If-Match: $OOSVER" -X POST "$SO/sales-orders/$OOSID/confirm")
ck "confirm OOS sales order -> 200" "$CODE" "200"
oos_cancelled=0
for i in $(seq 1 30); do
  ST=$(curl -sS "${AH[@]}" "$SO/sales-orders/$OOSID" | jq -r .status)
  [ "$ST" = "Cancelled" ] && oos_cancelled=1 && break
  sleep 2
done
ck "OOS sales order compensated -> Cancelled" "$oos_cancelled" "1"

echo "== 10. masterdata cache fan-out: party cached in all 5 downstream DBs (SAD 11.7) =="
# The party created in 5h (masterdata.party.upserted) is consumed independently by
# ap/ar/inventory/sales/procurement into each service's local masterdata_cache.
# Verified via psql (no HTTP read endpoint for the cache read-model).
md_ap=0; md_ar=0; md_inv=0; md_so=0; md_pr=0
mdq() { docker exec erp-slice-db-1 psql -U postgres -d "$1" -tAc "SELECT count(*) FROM $2.masterdata_cache WHERE party_id='$PARTYID';" 2>/dev/null | tr -d '[:space:]'; }
for i in $(seq 1 30); do
  [ "$(mdq ap_db ap)" = 1 ] && md_ap=1
  [ "$(mdq ar_db ar)" = 1 ] && md_ar=1
  [ "$(mdq inventory_db inv)" = 1 ] && md_inv=1
  [ "$(mdq sales_db so)" = 1 ] && md_so=1
  [ "$(mdq procurement_db pr)" = 1 ] && md_pr=1
  [ "$md_ap" = 1 ] && [ "$md_ar" = 1 ] && [ "$md_inv" = 1 ] && [ "$md_so" = 1 ] && [ "$md_pr" = 1 ] && break
  sleep 2
done
ck "ap cached masterdata party (fan-out)" "$md_ap" "1"
ck "ar cached masterdata party (fan-out)" "$md_ar" "1"
ck "inventory cached masterdata party (fan-out)" "$md_inv" "1"
ck "sales cached masterdata party (fan-out)" "$md_so" "1"
ck "procurement cached masterdata party (fan-out)" "$md_pr" "1"

echo "== 11. INT-002/003: treasury schedules payments from ap.invoice.recorded + payroll.run.completed =="
trs_ap=0; trs_pay=0
tq() { docker exec erp-slice-db-1 psql -U postgres -d treasury_db -tAc "SELECT count(*) FROM trs.payment WHERE payment_no='$1' AND status='Draft';" 2>/dev/null | tr -d '[:space:]'; }
for i in $(seq 1 30); do
  [ "$(tq "PAY-INV-$TS")" = 1 ] && trs_ap=1
  [ "$(tq "PAY-RUN-$TS")" = 1 ] && trs_pay=1
  [ "$trs_ap" = 1 ] && [ "$trs_pay" = 1 ] && break
  sleep 2
done
ck "treasury scheduled payment from ap.invoice.recorded" "$trs_ap" "1"
ck "treasury scheduled payment from payroll.run.completed" "$trs_pay" "1"

echo "== 12. INT-001: tax issues tax invoice (e-Faktur) from ar.invoice.issued =="
tax_ar=0
for i in $(seq 1 30); do
  [ "$(docker exec erp-slice-db-1 psql -U postgres -d tax_db -tAc "SELECT count(*) FROM tax.tax_transaction WHERE efaktur_no='EF-AR-$TS';" 2>/dev/null | tr -d '[:space:]')" = 1 ] && tax_ar=1 && break
  sleep 2
done
ck "tax issued tax invoice from ar.invoice.issued" "$tax_ar" "1"

echo "== 13. Currency Service & Gateway: List, Rate Creation, Triangulation Convert, SSE =="
SYS_ORG=00000000-0000-0000-0000-000000000000
SYS_AH=(-H "X-Organization-Id: $SYS_ORG" -H "X-User-Id: $USER" -H "X-Roles: SYSTEM_ADMIN" -H "Content-Type: application/json")
CUR_AH=(-H "X-Organization-Id: $ORGA,$SYS_ORG" -H "X-User-Id: $USER" -H "X-Roles: SYSTEM_ADMIN" -H "Content-Type: application/json")

# 13a. Direct port 8100 readyz & system org write protection
cur_ready=$(curl -s -o /dev/null -w "%{http_code}" "$CUR/readyz" || true)
ck "13a. currency-service direct /readyz (8100)" "$cur_ready" "200"
cur_sys_write=$(curl -s -o /dev/null -w "%{http_code}" "${SYS_AH[@]}" -H "Idempotency-Key: sys-block-$TS" -d '{"code":"ZZZ","name":"Blocked","precision":2}' "$CUR/currencies" || true)
ck "13a-sec. currency-service rejects write to system org -> 403" "$cur_sys_write" "403"

# Ensure base currencies (IDR, USD) and SPOT rate type exist in system org via DB seed if unseeded
docker exec erp-slice-db-1 psql -U postgres -d currency_db -q -v ON_ERROR_STOP=1 -c "
  INSERT INTO cur.currency (organization_id, code, numeric_code, name, precision, is_active, created_by) VALUES
    ('$SYS_ORG', 'IDR', '360', 'Indonesian Rupiah', 2, true, '$USER'),
    ('$SYS_ORG', 'USD', '840', 'US Dollar', 2, true, '$USER')
  ON CONFLICT (organization_id, code) DO NOTHING;
  INSERT INTO cur.exchange_rate_type (organization_id, code, name, created_by) VALUES
    ('$SYS_ORG', 'SPOT', 'Spot Rate', '$USER')
  ON CONFLICT (organization_id, code) DO NOTHING;
" >/dev/null 2>&1 || true

# 13b. Direct port 8100 list currencies (200 + includes IDR)
cur_list_res=$(curl -sS -w $'\n%{http_code}' "${CUR_AH[@]}" "$CUR/currencies?page=1&size=20" || true)
cur_list_code=$(printf '%s' "$cur_list_res" | tail -1)
cur_list_body=$(printf '%s' "$cur_list_res" | sed '$d')
ck "13b. currency-service direct /currencies list (8100)" "$cur_list_code" "200"
ckc "13b. currency-service /currencies includes IDR" "$cur_list_body" '"IDR"'

# 13c. Create/upsert USD -> IDR SPOT rate on $CUR/exchange-rates (201 or 200)
RATE_BODY='{"fromCurrency":"USD","toCurrency":"IDR","rateTypeId":"SPOT","rate":"15850.00000000","validFrom":"2026-03-15"}'
cur_rate_code=$(curl -s -o /dev/null -w "%{http_code}" "${CUR_AH[@]}" -H "Idempotency-Key: rate-$TS" -d "$RATE_BODY" "$CUR/exchange-rates" || true)
if [ "$cur_rate_code" = "409" ]; then
  EXISTING_RATE=$(curl -sS "${CUR_AH[@]}" "$CUR/exchange-rates?from=USD&to=IDR&onDate=2026-03-15" || echo "{}")
  RATE_ID=$(printf '%s' "$EXISTING_RATE" | jq -r '[.data[] | select(.validFrom=="2026-03-15" and .rateTypeCode=="SPOT")][0].id // .data[0].id // empty')
  RATE_VER=$(printf '%s' "$EXISTING_RATE" | jq -r '[.data[] | select(.validFrom=="2026-03-15" and .rateTypeCode=="SPOT")][0].rowVersion // .data[0].rowVersion // 1')
  if [ -n "$RATE_ID" ]; then
    cur_rate_code=$(curl -s -o /dev/null -w "%{http_code}" "${CUR_AH[@]}" -H "If-Match: $RATE_VER" -X PUT -d '{"rate":"15850.00000000"}' "$CUR/exchange-rates/$RATE_ID" || true)
  fi
fi
[ "$cur_rate_code" = "201" ] && cur_rate_code="200"
ck "13c. currency-service create/upsert USD->IDR SPOT rate (8100)" "$cur_rate_code" "200"

# 13d. Direct port 8100 convert 100.0000 USD -> IDR
CONV_BODY='{"fromCurrency":"USD","toCurrency":"IDR","rateTypeCode":"SPOT","amount":"100.0000","onDate":"2026-03-15"}'
cur_conv_res=$(curl -sS -w $'\n%{http_code}' "${CUR_AH[@]}" -d "$CONV_BODY" "$CUR/exchange-rates/convert" || true)
cur_conv_code=$(printf '%s' "$cur_conv_res" | tail -1)
cur_conv_body=$(printf '%s' "$cur_conv_res" | sed '$d')
ck "13d. currency-service direct /exchange-rates/convert 100.0000 USD->IDR (8100)" "$cur_conv_code" "200"
ckc "13d. currency-service convert result has convertedAmount" "$cur_conv_body" '"convertedAmount"'

# 13e. Authenticate against API Gateway ($GW/auth/login) and list currencies via Gateway
GW_LOGIN=$(curl -sS -H "Content-Type: application/json" -d '{"username":"admin.finance","password":"dev","organizationId":"'"$ORGA"'"}' "$GW/auth/login" || echo "{}")
GW_TOKEN=$(printf '%s' "$GW_LOGIN" | jq -r '.token // empty')
GW_TICKET=$(printf '%s' "$GW_LOGIN" | jq -r '.streamTicket // empty')
GW_AH=(-H "Authorization: Bearer $GW_TOKEN" -H "X-Organization-Id: $ORGA" -H "Content-Type: application/json")
gw_cur_code=$(curl -s -o /dev/null -w "%{http_code}" "${GW_AH[@]}" "$GW/currencies?page=1&size=5" || true)
ck "13e. api-gateway authenticated /currencies (8000)" "$gw_cur_code" "200"

# 13f. Gateway routing for POST /exchange-rates/convert with Bearer token
gw_conv_code=$(curl -s -o /dev/null -w "%{http_code}" "${GW_AH[@]}" -d "$CONV_BODY" "$GW/exchange-rates/convert" || true)
ck "13f. api-gateway authenticated /exchange-rates/convert (8000)" "$gw_conv_code" "200"

# 13g. Gateway SSE event stream with Bearer token & streamTicket
gw_sse_code=$(curl -s -N --max-time 2 -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $GW_TOKEN" "$GW/stream/events?ticket=$GW_TICKET" || true)
ck "13g. api-gateway authenticated SSE stream (8000)" "$gw_sse_code" "200"

echo "== 14. Dimension & Localization Services (SVC-021, SVC-029) =="
# 14a. readiness (direct ports 8102 / 8110)
ck "14a. dimension-service direct /readyz (8102)" "$(curl -s -o /dev/null -w '%{http_code}' "$DIM/readyz" || true)" "200"
ck "14a. localization-service direct /readyz (8110)" "$(curl -s -o /dev/null -w '%{http_code}' "$LOC/readyz" || true)" "200"

# 14b. cost centers + profit centers
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: cc-$TS" -d '{"code":"CC-'"$TS"'","name":"Operasional","effectiveFrom":"2026-01-01","status":"Active"}' "$DIM/cost-centers")
CODE=$(printf '%s' "$R" | tail -1); BODY=$(printf '%s' "$R" | sed '$d')
ck "14b. create cost center -> 201" "$CODE" "201"
CCID=$(printf '%s' "$BODY" | jq -r .id)
R=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: cc2-$TS" -d '{"code":"CC2-'"$TS"'","name":"Pemasaran","effectiveFrom":"2026-01-01","status":"Active"}' "$DIM/cost-centers")
CC2ID=$(printf '%s' "$R" | sed '$d' | jq -r .id)
CODE=$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -H "Idempotency-Key: pc-$TS" -d '{"code":"PC-'"$TS"'","name":"Divisi A","effectiveFrom":"2026-01-01"}' "$DIM/profit-centers")
ck "14b. create profit center -> 201" "$CODE" "201"
CODE=$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -H "Idempotency-Key: cc-$TS" -d '{"code":"CC-'"$TS"'","name":"Operasional","effectiveFrom":"2026-01-01","status":"Active"}' "$DIM/cost-centers")
ck "14b. cost center idempotent replay -> 201" "$CODE" "201"

# 14c. RLS: org B cannot read org A cost center
ck "14c. dimension RLS org B get -> 404" "$(curl -sS -o /dev/null -w '%{http_code}' -H "X-Organization-Id: $ORGB" "$DIM/cost-centers/$CCID")" "404"

# 14d. allocation rule BR: totals must be exactly 100 (422) and valid rule (201)
BADALLOC='{"sourceCostCenterId":"'"$CCID"'","name":"Bad","allocationBasis":"headcount","targets":[{"costCenterId":"'"$CC2ID"'","percentage":"99.99"}]}'
ck "14d. allocation total != 100 -> 422" "$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -d "$BADALLOC" "$DIM/allocation-rules")" "422"
OKALLOC='{"sourceCostCenterId":"'"$CCID"'","name":"Ok","allocationBasis":"headcount","targets":[{"costCenterId":"'"$CC2ID"'","percentage":"100"}]}'
ck "14d. allocation total = 100 -> 201" "$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -H "Idempotency-Key: alloc-$TS" -d "$OKALLOC" "$DIM/allocation-rules")" "201"

# 14e. localization: seeded languages visible (system org) + translation bundle with fallback overlay
LANGS=$(curl -sS "${AH[@]}" "$LOC/languages?size=50")
ckc "14e. localization seeded language 'en' visible" "$LANGS" '"en"'
ckc "14e. localization seeded language 'id' visible" "$LANGS" '"id"'
CODE=$(curl -sS -o /dev/null -w '%{http_code}' "${AH[@]}" -H "Idempotency-Key: tr-$TS" -d '{"languageCode":"id","resourceKey":"smoke.'"$TS"'","resourceValue":"Halo","module":"smoke"}' "$LOC/translations")
ck "14e. create translation -> 201" "$CODE" "201"
BUNDLE=$(curl -sS "${AH[@]}" "$LOC/translation-bundles/id")
ckc "14e. translation bundle contains new key" "$BUNDLE" "smoke.$TS"

# 14f. gateway routing + RBAC
GW14=$(curl -s -o /dev/null -w "%{http_code}" "${GW_AH[@]}" "$GW/cost-centers?page=1&size=5" || true)
ck "14f. api-gateway authenticated /cost-centers (8000)" "$GW14" "200"
GW14B=$(curl -s -o /dev/null -w "%{http_code}" "${GW_AH[@]}" "$GW/translation-bundles/id" || true)
ck "14f. api-gateway authenticated /translation-bundles/id (8000)" "$GW14B" "200"

# 14g. audit consumed dimension/localization domain events
a_dim=0; a_loc=0
for i in $(seq 1 20); do
  A=$(curl -sS -H "X-Organization-Id: $ORGA" "$AUD/audit-events?size=500")
  printf '%s' "$A" | jq -e '.data[] | select(.action=="dimension.cost_center.upserted" and .entityId=="'"$CCID"'")' >/dev/null 2>&1 && a_dim=1
  printf '%s' "$A" | jq -e '.data[] | select(.action=="localization.bundle.updated")' >/dev/null 2>&1 && a_loc=1
  [ "$a_dim" = 1 ] && [ "$a_loc" = 1 ] && break
  sleep 2
done
ck "14g. audit consumed dimension.cost_center.upserted" "$a_dim" "1"
ck "14g. audit consumed localization.bundle.updated" "$a_loc" "1"

echo "== 15. Extended Microservices (fixedasset, budget, financialclose, consolidation, project, asset, maintenance, crm) =="
for pair in "fixedasset:$FA" "budget:$BDG" "financialclose:$FCL" "consolidation:$CON" "project:$PRJ" "asset:$AST" "maintenance:$MNT" "crm:$CRM"; do
  svc="${pair%%:*}"; url="${pair#*:}"
  ck "15a. $svc /readyz" "$(curl -s -o /dev/null -w '%{http_code}' "$url/readyz" || true)" "200"
done

# 15c. End-to-end POST + idempotent replay across the 8 extended services
FA_CAT_BODY='{"code":"FAC-'"$TS"'","name":"Kategori Smoke","defaultDepreciationMethod":"StraightLine","defaultUsefulLifeMonths":48}'
ck "15c. fixedasset create category -> 201" "$(curl -s -o /dev/null -w '%{http_code}' "${AH[@]}" -H "Idempotency-Key: fac-$TS" -d "$FA_CAT_BODY" "$FA/asset-categories" || true)" "201"
ck "15c. fixedasset create category idempotent replay -> 201" "$(curl -s -o /dev/null -w '%{http_code}' "${AH[@]}" -H "Idempotency-Key: fac-$TS" -d "$FA_CAT_BODY" "$FA/asset-categories" || true)" "201"

BDG_BODY='{"code":"BDG-'"$TS"'","name":"Anggaran Smoke","financialOrganizationRef":"'"$FINORG"'","fiscalYearRef":"'"$PERIOD"'","version":1}'
ck "15c. budget create -> 201" "$(curl -s -o /dev/null -w '%{http_code}' "${AH[@]}" -H "Idempotency-Key: bdg-$TS" -d "$BDG_BODY" "$BDG/budgets" || true)" "201"
ck "15c. budget create idempotent replay -> 201" "$(curl -s -o /dev/null -w '%{http_code}' "${AH[@]}" -H "Idempotency-Key: bdg-$TS" -d "$BDG_BODY" "$BDG/budgets" || true)" "201"

FCL_BODY='{"code":"FCL-'"$TS"'","name":"Template Close Smoke","financialOrganizationRef":"'"$FINORG"'","tasks":[{"taskOrder":1,"taskName":"Recon"}]}'
ck "15c. financialclose create template -> 201" "$(curl -s -o /dev/null -w '%{http_code}' "${AH[@]}" -H "Idempotency-Key: fcl-$TS" -d "$FCL_BODY" "$FCL/closing-templates" || true)" "201"

CON_BODY='{"code":"GRP-'"$TS"'","name":"Grup Smoke","presentationCurrencyCode":"IDR"}'
ck "15c. consolidation create group -> 201" "$(curl -s -o /dev/null -w '%{http_code}' "${AH[@]}" -H "Idempotency-Key: con-$TS" -d "$CON_BODY" "$CON/consolidation-groups" || true)" "201"

PRJ_BODY='{"code":"PRJ-'"$TS"'","name":"Proyek Smoke","projectType":"Capital","currencyCode":"IDR"}'
ck "15c. project create -> 201" "$(curl -s -o /dev/null -w '%{http_code}' "${AH[@]}" -H "Idempotency-Key: prj-$TS" -d "$PRJ_BODY" "$PRJ/projects" || true)" "201"

AST_RES=$(curl -sS -w $'\n%{http_code}' "${AH[@]}" -H "Idempotency-Key: ast-$TS" -d '{"assetCode":"AST-'"$TS"'","name":"Unit Smoke","category":"HVAC"}' "$AST/assets" || true)
AST_CODE=$(printf '%s' "$AST_RES" | tail -1)
AST_ID=$(printf '%s' "$AST_RES" | sed '$d' | jq -r '.id // empty')
ck "15c. asset create -> 201" "$AST_CODE" "201"

MNT_ASSET_ID="${AST_ID:-a55e0000-0000-4000-8000-000000000001}"
ck "15c. maintenance create work order -> 201" "$(curl -s -o /dev/null -w '%{http_code}' "${AH[@]}" -H "Idempotency-Key: mnt-$TS" -d '{"woNo":"MWO-'"$TS"'","assetRef":"'"$MNT_ASSET_ID"'","maintenanceType":"Preventive"}' "$MNT/maintenance-work-orders" || true)" "201"

ck "15c. crm create lead -> 201" "$(curl -s -o /dev/null -w '%{http_code}' "${AH[@]}" -H "Idempotency-Key: crm-$TS" -d '{"code":"LEAD-'"$TS"'","name":"Prospek Smoke","estimatedValue":"25000000.0000"}' "$CRM/leads" || true)" "201"

ck "15b. gw /fixed-assets" "$(curl -s -o /dev/null -w '%{http_code}' "${GW_AH[@]}" "$GW/fixed-assets?size=5" || true)" "200"
ck "15b. gw /budgets" "$(curl -s -o /dev/null -w '%{http_code}' "${GW_AH[@]}" "$GW/budgets?size=5" || true)" "200"
ck "15b. gw /closing-runs" "$(curl -s -o /dev/null -w '%{http_code}' "${GW_AH[@]}" "$GW/closing-runs?size=5" || true)" "200"
ck "15b. gw /consolidation-runs" "$(curl -s -o /dev/null -w '%{http_code}' "${GW_AH[@]}" "$GW/consolidation-runs?size=5" || true)" "200"
ck "15b. gw /projects" "$(curl -s -o /dev/null -w '%{http_code}' "${GW_AH[@]}" "$GW/projects?size=5" || true)" "200"
ck "15b. gw /assets" "$(curl -s -o /dev/null -w '%{http_code}' "${GW_AH[@]}" "$GW/assets?size=5" || true)" "200"
ck "15b. gw /maintenance-work-orders" "$(curl -s -o /dev/null -w '%{http_code}' "${GW_AH[@]}" "$GW/maintenance-work-orders?size=5" || true)" "200"
ck "15b. gw /opportunities" "$(curl -s -o /dev/null -w '%{http_code}' "${GW_AH[@]}" "$GW/opportunities?size=5" || true)" "200"

echo ""
echo "==================== RESULT: $pass passed, $fail failed ===================="
[ "$fail" = 0 ] && echo "E2E_SMOKE_PASS" || echo "E2E_SMOKE_FAIL"
exit "$fail"
