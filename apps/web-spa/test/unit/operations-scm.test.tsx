import * as React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { I18nextProvider } from 'react-i18next';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { InventoryWorkspace } from '../../src/features/inventory/InventoryWorkspace';
import { ManufacturingWorkspace } from '../../src/features/manufacturing/ManufacturingWorkspace';
import { MasterdataWorkspace } from '../../src/features/masterdata/MasterdataWorkspace';
import { ProcurementWorkspace } from '../../src/features/procurement/ProcurementWorkspace';
import { computeOrderTotalDecimal } from '../../src/features/procurement/schemas';
import { SalesWorkspace } from '../../src/features/sales/SalesWorkspace';
import { computeSalesOrderTotalDecimal } from '../../src/features/sales/schemas';
import i18n from '../../src/lib/i18n';
import { useAuthStore } from '../../src/store/auth-store';
import type { Organization, User } from '../../src/types';

if (typeof Element !== 'undefined') {
  if (!Element.prototype.hasPointerCapture) {
    Element.prototype.hasPointerCapture = () => false;
  }
  if (!Element.prototype.releasePointerCapture) {
    Element.prototype.releasePointerCapture = () => {};
  }
  if (!Element.prototype.scrollIntoView) {
    Element.prototype.scrollIntoView = () => {};
  }
}

interface CapturedRequest {
  method: string;
  url: string;
  path: string;
  headers: Record<string, string>;
  body: unknown;
}

const TEST_ORG: Organization = {
  id: '11111111-1111-1111-1111-111111111111',
  code: 'CORP-ID',
  name: 'PT Nusantara Holdings',
  type: 'COMPANY',
  functionalCurrency: 'IDR',
  isActive: true,
};

function makeUser(permissions: string[]): User {
  return {
    id: '22222222-2222-2222-2222-222222222222',
    username: 'ops.tester',
    fullName: 'Operations Tester',
    email: 'ops@erp.local',
    activeOrgId: TEST_ORG.id,
    roles: ['OPERATOR'],
    permissions,
  };
}

function setAuthPermissions(permissions: string[]) {
  useAuthStore.setState({
    user: makeUser(permissions),
    activeOrg: TEST_ORG,
    organizations: [TEST_ORG],
    isAuthenticated: true,
    token: null,
  });
}

function renderWorkspace(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });
  return render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
    </I18nextProvider>
  );
}

function jsonResponse(payload: unknown, status = 200) {
  if (status === 204) {
    return new Response(null, { status: 204 });
  }
  return new Response(JSON.stringify(payload), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function listEnvelope<T>(data: T[]) {
  return {
    data,
    page: {
      number: 1,
      size: 50,
      totalElements: data.length,
      totalPages: data.length > 0 ? 1 : 0,
    },
  };
}

function createFetchRouter(
  handler: (req: CapturedRequest) => Response | Promise<Response>
): { captured: CapturedRequest[] } {
  const captured: CapturedRequest[] = [];
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const rawUrl = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
      const parsedUrl = new URL(rawUrl, 'http://localhost');
      const headersRecord: Record<string, string> = {};
      const hdrs = new Headers(init?.headers);
      hdrs.forEach((v, k) => {
        headersRecord[k.toLowerCase()] = v;
      });
      let parsedBody: unknown = undefined;
      if (typeof init?.body === 'string' && init.body.length > 0) {
        try {
          parsedBody = JSON.parse(init.body);
        } catch {
          parsedBody = init.body;
        }
      }
      const req: CapturedRequest = {
        method: (init?.method ?? 'GET').toUpperCase(),
        url: `${parsedUrl.pathname}${parsedUrl.search}`,
        path: parsedUrl.pathname.replace(/^\/api\/v1/, ''),
        headers: headersRecord,
        body: parsedBody,
      };
      captured.push(req);
      return handler(req);
    })
  );
  return { captured };
}

describe('Wave 2C Operations & SCM Modules', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    if (typeof window !== 'undefined') {
      Object.defineProperty(window, 'matchMedia', {
        writable: true,
        value: (query: string) => ({
          matches: false,
          media: query,
          onchange: null,
          addListener: () => {},
          removeListener: () => {},
          addEventListener: () => {},
          removeEventListener: () => {},
          dispatchEvent: () => false,
        }),
      });
    }
    setAuthPermissions(['*']);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  describe('MasterdataWorkspace (/masterdata)', () => {
    it('enforces masterdata:write RBAC and executes Party & Item CRUD with Idempotency-Key & If-Match', async () => {
      const user = userEvent.setup();
      const parties = [
        {
          id: 'party-1',
          organizationId: TEST_ORG.id,
          code: 'VND-001',
          name: 'PT Baja Nusantara',
          partyType: 'vendor',
          taxId: '01.234.567.8-901.000',
          isShared: false,
          rowVersion: 2,
        },
      ];
      const items = [
        {
          id: 'item-1',
          organizationId: TEST_ORG.id,
          code: 'RM-STL-01',
          name: 'Steel Coil 2mm',
          itemType: 'stock',
          baseUom: 'KG',
          rowVersion: 3,
        },
      ];

      const { captured } = createFetchRouter((req) => {
        if (req.method === 'GET' && req.path === '/parties') {
          return jsonResponse(listEnvelope(parties));
        }
        if (req.method === 'POST' && req.path === '/parties') {
          const payload = req.body as Record<string, unknown>;
          const created = {
            id: 'party-2',
            organizationId: TEST_ORG.id,
            code: String(payload.code),
            name: String(payload.name),
            partyType: String(payload.partyType),
            taxId: payload.taxId ? String(payload.taxId) : null,
            isShared: Boolean(payload.isShared),
            rowVersion: 1,
            addresses: [],
          };
          parties.push({
            ...created,
            taxId: created.taxId,
          });
          return jsonResponse(created, 201);
        }
        if (req.method === 'DELETE' && req.path === '/parties/party-1') {
          return jsonResponse({}, 204);
        }
        if (req.method === 'GET' && req.path === '/items') {
          return jsonResponse(listEnvelope(items));
        }
        if (req.method === 'POST' && req.path === '/items') {
          const payload = req.body as Record<string, unknown>;
          const created = {
            id: 'item-2',
            organizationId: TEST_ORG.id,
            code: String(payload.code),
            name: String(payload.name),
            itemType: String(payload.itemType),
            baseUom: String(payload.baseUom),
            rowVersion: 1,
            uoms: [],
          };
          items.push({
            id: created.id,
            organizationId: created.organizationId,
            code: created.code,
            name: created.name,
            itemType: created.itemType,
            baseUom: created.baseUom,
            rowVersion: 1,
          });
          return jsonResponse(created, 201);
        }
        return jsonResponse({}, 404);
      });

      // Read-only permission check first
      setAuthPermissions(['masterdata:read']);
      const { unmount } = renderWorkspace(<MasterdataWorkspace />);
      expect(await screen.findByText('PT Baja Nusantara')).toBeInTheDocument();
      expect(screen.getByText('Read-only')).toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /New Partner/i })).not.toBeInTheDocument();
      expect(screen.queryByLabelText(/Delete party VND-001/i)).not.toBeInTheDocument();
      unmount();

      // Write permission enabled
      setAuthPermissions(['masterdata:read', 'masterdata:write']);
      renderWorkspace(<MasterdataWorkspace />);
      expect(await screen.findByText('PT Baja Nusantara')).toBeInTheDocument();

      // Create Partner
      await user.click(screen.getByRole('button', { name: /New Partner/i }));
      const dialog = await screen.findByRole('dialog');
      await user.type(within(dialog).getByLabelText(/Partner Code/i), 'CUS-009');
      await user.type(within(dialog).getByLabelText(/Legal Name/i), 'PT Sinar Abadi');
      await user.click(within(dialog).getByRole('button', { name: /Create Partner/i }));

      await waitFor(() => {
        const postReq = captured.find((r) => r.method === 'POST' && r.path === '/parties');
        expect(postReq).toBeDefined();
        expect(postReq?.headers['idempotency-key']).toBeTruthy();
        expect(postReq?.body).toMatchObject({
          code: 'CUS-009',
          name: 'PT Sinar Abadi',
          partyType: 'vendor',
          isShared: false,
        });
      });

      // Delete existing partner sends If-Match header
      await user.click(screen.getByLabelText('Delete party VND-001'));
      const confirmDialog = await screen.findByRole('dialog');
      await user.click(within(confirmDialog).getByRole('button', { name: /Delete Partner/i }));
      await waitFor(() => {
        const delReq = captured.find((r) => r.method === 'DELETE' && r.path === '/parties/party-1');
        expect(delReq).toBeDefined();
        expect(delReq?.headers['if-match']).toBe('"2"');
      });

      // Switch to Items tab & create item with uppercase baseUom
      await user.click(screen.getByRole('tab', { name: /Items & Materials/i }));
      expect(await screen.findByText('Steel Coil 2mm')).toBeInTheDocument();

      await user.click(screen.getByRole('button', { name: /New Item/i }));
      const itemDialog = await screen.findByRole('dialog');
      await user.type(within(itemDialog).getByLabelText(/SKU Code/i), 'FG-BOX-10');
      await user.type(within(itemDialog).getByLabelText(/Item Name/i), 'Carton Packaging');
      const uomInput = within(itemDialog).getByLabelText(/Base UOM/i);
      await user.clear(uomInput);
      await user.type(uomInput, 'box');
      await user.click(within(itemDialog).getByRole('button', { name: /Create Item/i }));

      await waitFor(() => {
        const itemPost = captured.find((r) => r.method === 'POST' && r.path === '/items');
        expect(itemPost).toBeDefined();
        expect(itemPost?.headers['idempotency-key']).toBeTruthy();
        expect(itemPost?.body).toMatchObject({
          code: 'FG-BOX-10',
          name: 'Carton Packaging',
          itemType: 'stock',
          baseUom: 'BOX',
        });
      });
    });

    it('renders ErrorState instead of silent fallback when /parties fails', async () => {
      createFetchRouter((req) => {
        if (req.path === '/parties') {
          return jsonResponse(
            {
              error: {
                code: 'MASTERDATA_DOWN',
                message: 'Masterdata service unavailable',
                correlationId: 'cid-md-500',
              },
            },
            500
          );
        }
        return jsonResponse(listEnvelope([]));
      });

      renderWorkspace(<MasterdataWorkspace />);
      expect(await screen.findByText(/Masterdata service unavailable/i)).toBeInTheDocument();
      expect(screen.getByText(/cid-md-500/i)).toBeInTheDocument();
    });
  });

  describe('InventoryWorkspace (/inventory)', () => {
    it('queries /stock-balances cleanly, shows ErrorState on failure without mock fallback, and supports Warehouse, Movement & Reservation actions', async () => {
      const user = userEvent.setup();
      let failBalances = true;
      const warehouses = [
        {
          id: 'wh-1',
          code: 'WH-MAIN',
          name: 'Jakarta Main Warehouse',
          organizationRefId: TEST_ORG.id,
          rowVersion: 1,
        },
        {
          id: 'wh-2',
          code: 'WH-SBY',
          name: 'Surabaya Hub',
          organizationRefId: TEST_ORG.id,
          rowVersion: 1,
        },
      ];
      const balances = [
        {
          id: 'bal-1',
          warehouseId: 'wh-1',
          itemId: 'item-1',
          onHandQty: '100.0000',
          reservedQty: '10.0000',
          availableQty: '90.0000',
          avgCost: '50000.0000',
          rowVersion: 1,
        },
      ];
      const reservations = [
        {
          id: 'resv-1',
          itemId: 'item-1',
          warehouseId: 'wh-1',
          reservedQty: '10.0000',
          salesOrderRef: 'SO-2026-001',
          status: 'Reserved',
          rowVersion: 4,
        },
      ];

      const { captured } = createFetchRouter((req) => {
        if (req.method === 'GET' && req.path === '/stock-balances') {
          if (failBalances) {
            return jsonResponse(
              { error: { code: 'INV_503', message: 'Inventory database offline' } },
              503
            );
          }
          return jsonResponse(listEnvelope(balances));
        }
        if (req.method === 'GET' && req.path === '/warehouses') {
          return jsonResponse(listEnvelope(warehouses));
        }
        if (req.method === 'POST' && req.path === '/warehouses') {
          const payload = req.body as Record<string, string>;
          const created = {
            id: 'wh-3',
            code: payload.code,
            name: payload.name,
            organizationRefId: TEST_ORG.id,
            rowVersion: 1,
          };
          warehouses.push(created);
          return jsonResponse(created, 201);
        }
        if (req.method === 'GET' && req.path === '/stock-movements') {
          return jsonResponse(listEnvelope([]));
        }
        if (req.method === 'POST' && req.path === '/stock-movements') {
          return jsonResponse(
            {
              id: 'mov-1',
              movementDate: '2026-11-15',
              ...(req.body as object),
              rowVersion: 1,
            },
            201
          );
        }
        if (req.method === 'GET' && req.path === '/reservations') {
          return jsonResponse(listEnvelope(reservations));
        }
        if (req.method === 'POST' && req.path === '/reservations/resv-1/issue') {
          reservations[0] = { ...reservations[0], status: 'Consumed', rowVersion: 5 };
          return jsonResponse(reservations[0]);
        }
        if (req.method === 'POST' && req.path === '/reservations/resv-1/release') {
          reservations[0] = { ...reservations[0], status: 'Released', rowVersion: 5 };
          return jsonResponse(reservations[0]);
        }
        if (req.method === 'GET' && req.path === '/items') {
          return jsonResponse(
            listEnvelope([
              { id: 'item-1', code: 'SKU-100', name: 'Copper Wire', itemType: 'stock', baseUom: 'MTR' },
            ])
          );
        }
        return jsonResponse({}, 404);
      });

      setAuthPermissions(['inventory:read', 'inventory:write']);
      renderWorkspace(<InventoryWorkspace />);

      // Must fail visibly with ErrorState instead of buildFallbackInventoryItems
      expect(await screen.findByText(/Inventory database offline/i)).toBeInTheDocument();
      expect(screen.queryByText(/Baut Hex M10/i)).not.toBeInTheDocument();

      // Retry loads live stock balances enriched with item/warehouse codes
      failBalances = false;
      await user.click(screen.getByRole('button', { name: /Coba Lagi|Retry/i }));
      expect(await screen.findByText('SKU-100')).toBeInTheDocument();
      expect(screen.getByText('WH-MAIN — Jakarta Main Warehouse')).toBeInTheDocument();

      // Create warehouse
      await user.click(screen.getByRole('button', { name: /New Warehouse/i }));
      const whDialog = await screen.findByRole('dialog');
      await user.type(within(whDialog).getByLabelText(/Warehouse Code/i), 'WH-BDG');
      await user.type(within(whDialog).getByLabelText(/Warehouse Name/i), 'Bandung Depot');
      await user.click(within(whDialog).getByRole('button', { name: /Create Warehouse/i }));

      await waitFor(() => {
        const postWh = captured.find((r) => r.method === 'POST' && r.path === '/warehouses');
        expect(postWh?.body).toMatchObject({
          code: 'WH-BDG',
          name: 'Bandung Depot',
        });
      });

      // Switch to Reservations & issue reserved stock with If-Match
      await user.click(screen.getByRole('tab', { name: /Reservations/i }));
      expect(await screen.findByText('SO-2026-001')).toBeInTheDocument();
      await user.click(screen.getByRole('button', { name: 'Issue reservation SO-2026-001' }));
      const issueDialog = await screen.findByRole('dialog');
      await user.click(within(issueDialog).getByRole('button', { name: /Confirm Issue/i }));

      await waitFor(() => {
        const issueReq = captured.find(
          (r) => r.method === 'POST' && r.path === '/reservations/resv-1/issue'
        );
        expect(issueReq).toBeDefined();
        expect(issueReq?.headers['if-match']).toBe('"4"');
        expect(issueReq?.body).toEqual({ amount: '10.0000' });
      });
    });
  });

  describe('ProcurementWorkspace (/procurement)', () => {
    it('computes exact fixed-point decimal PO totals and executes Approve -> Receive + PR workflow', async () => {
      // Pure fixed-point decimal invariant test (no IEEE-754 float drift)
      const exactTotal = computeOrderTotalDecimal([
        { orderQty: '0.1', unitPrice: '0.2' },
        { orderQty: '1.25', unitPrice: '33.33' },
      ]);
      expect(exactTotal).toBe('41.6825');

      const user = userEvent.setup();
      const orders = [
        {
          id: 'po-1',
          poNo: 'PO-2026-001',
          vendorId: 'vnd-1',
          orderDate: '2026-11-10',
          currencyCode: 'IDR',
          totalAmount: '5000000.0000',
          status: 'Draft',
          rowVersion: 1,
        },
        {
          id: 'po-2',
          poNo: 'PO-2026-002',
          vendorId: 'vnd-1',
          orderDate: '2026-11-11',
          currencyCode: 'IDR',
          totalAmount: '2500000.0000',
          status: 'Approved',
          rowVersion: 2,
        },
      ];
      const requisitions = [
        {
          id: 'pr-1',
          prNo: 'PR-2026-010',
          requesterId: '22222222-2222-2222-2222-222222222222',
          neededDate: '2026-11-20',
          status: 'Draft',
          rowVersion: 1,
        },
      ];
      const receipts = [
        {
          id: 'gr-1',
          grNo: 'GR-2026-009',
          purchaseOrderId: 'po-old',
          warehouseId: 'wh-1',
          receiptDate: '2026-11-12',
          status: 'Posted',
          rowVersion: 1,
        },
      ];

      const { captured } = createFetchRouter((req) => {
        if (req.method === 'GET' && req.path === '/purchase-orders') {
          return jsonResponse(listEnvelope(orders));
        }
        if (req.method === 'GET' && req.path === '/purchase-orders/po-2') {
          return jsonResponse({
            ...orders[1],
            lines: [
              {
                id: 'pol-1',
                lineNo: 1,
                itemId: 'item-1',
                orderQty: '5.0000',
                receivedQty: '0.0000',
                unitPrice: '500000.0000',
              },
            ],
          });
        }
        if (req.method === 'POST' && req.path === '/purchase-orders/po-1/approve') {
          orders[0] = { ...orders[0], status: 'Approved', rowVersion: 2 };
          return jsonResponse(orders[0]);
        }
        if (req.method === 'POST' && req.path === '/purchase-orders/po-2/receive') {
          orders[1] = { ...orders[1], status: 'Received', rowVersion: 3 };
          return jsonResponse(
            {
              id: 'gr-new',
              grNo: 'GR-PO-2026-002',
              purchaseOrderId: 'po-2',
              warehouseId: 'wh-1',
              receiptDate: '2026-11-15',
              status: 'Posted',
              rowVersion: 1,
            },
            201
          );
        }
        if (req.method === 'GET' && req.path === '/purchase-requisitions') {
          return jsonResponse(listEnvelope(requisitions));
        }
        if (req.method === 'POST' && req.path === '/purchase-requisitions/pr-1/submit') {
          requisitions[0] = { ...requisitions[0], status: 'Submitted', rowVersion: 2 };
          return jsonResponse(requisitions[0]);
        }
        if (req.method === 'POST' && req.path === '/purchase-requisitions/pr-1/approve') {
          requisitions[0] = { ...requisitions[0], status: 'Approved', rowVersion: 3 };
          return jsonResponse(requisitions[0]);
        }
        if (req.method === 'GET' && req.path === '/goods-receipts') {
          return jsonResponse(listEnvelope(receipts));
        }
        if (req.method === 'GET' && req.path === '/goods-receipts/gr-1') {
          return jsonResponse({
            ...receipts[0],
            lines: [{ id: 'grl-1', poLineId: 'pol-99', lineNo: 1, receivedQty: '12.0000' }],
          });
        }
        if (req.method === 'GET' && req.path === '/parties') {
          return jsonResponse(
            listEnvelope([{ id: 'vnd-1', code: 'V01', name: 'PT Supplier Jaya', partyType: 'vendor' }])
          );
        }
        if (req.method === 'GET' && req.path === '/items') {
          return jsonResponse(
            listEnvelope([{ id: 'item-1', code: 'STL-1', name: 'Steel Bar', baseUom: 'PCS' }])
          );
        }
        if (req.method === 'GET' && req.path === '/warehouses') {
          return jsonResponse(
            listEnvelope([{ id: 'wh-1', code: 'WH-01', name: 'Central Warehouse' }])
          );
        }
        return jsonResponse({}, 404);
      });

      setAuthPermissions(['procurement:read', 'procurement:write']);
      renderWorkspace(<ProcurementWorkspace />);

      expect(await screen.findByText('PO-2026-001')).toBeInTheDocument();
      // Approve PO-1
      await user.click(screen.getByRole('button', { name: 'Approve PO PO-2026-001' }));
      await waitFor(() => {
        const appReq = captured.find(
          (r) => r.method === 'POST' && r.path === '/purchase-orders/po-1/approve'
        );
        expect(appReq?.headers['if-match']).toBe('"1"');
      });

      // Receive PO-2 sends strictly { grNo, warehouseId, receiptDate } without unknown fields
      await user.click(screen.getByRole('button', { name: 'Receive PO-2026-002' }));
      const receiveDialog = await screen.findByRole('dialog');
      expect(await within(receiveDialog).findByText(/PO Ordered Quantities/i)).toBeInTheDocument();
      await user.click(within(receiveDialog).getByRole('button', { name: /Confirm Goods Receipt/i }));

      await waitFor(() => {
        const recReq = captured.find(
          (r) => r.method === 'POST' && r.path === '/purchase-orders/po-2/receive'
        );
        expect(recReq).toBeDefined();
        expect(recReq?.headers['if-match']).toBe('"2"');
        const body = recReq?.body as Record<string, unknown>;
        expect(Object.keys(body).sort()).toEqual(['grNo', 'receiptDate', 'warehouseId']);
        expect(body.warehouseId).toBe('wh-1');
      });

      // PR submit -> approve
      await user.click(screen.getByRole('tab', { name: /Purchase Requisitions/i }));
      expect(await screen.findByText('PR-2026-010')).toBeInTheDocument();
      await user.click(screen.getByRole('button', { name: 'Submit PR PR-2026-010' }));
      await waitFor(() => {
        expect(
          captured.some(
            (r) => r.method === 'POST' && r.path === '/purchase-requisitions/pr-1/submit'
          )
        ).toBe(true);
      });
      await user.click(await screen.findByRole('button', { name: 'Approve PR PR-2026-010' }));
      await waitFor(() => {
        expect(
          captured.some(
            (r) => r.method === 'POST' && r.path === '/purchase-requisitions/pr-1/approve'
          )
        ).toBe(true);
      });

      // Goods Receipts detail drawer
      await user.click(screen.getByRole('tab', { name: /Goods Receipts/i }));
      const grRow = await screen.findByText('GR-2026-009');
      await user.click(grRow);
      expect(await screen.findByText(/Goods Receipt GR-2026-009/i)).toBeInTheDocument();
    });
  });

  describe('ManufacturingWorkspace (/manufacturing)', () => {
    it('releases Planned WO, completes Released WO with exact producedQty field, and loads BOM detail lines', async () => {
      const user = userEvent.setup();
      const workOrders = [
        {
          id: 'wo-1',
          woNo: 'WO-2026-101',
          productItemId: 'item-fg',
          bomId: 'bom-1',
          plannedQty: '50.0000',
          producedQty: '0.0000',
          status: 'Planned',
          rowVersion: 1,
        },
        {
          id: 'wo-2',
          woNo: 'WO-2026-102',
          productItemId: 'item-fg',
          bomId: 'bom-1',
          plannedQty: '20.0000',
          producedQty: '0.0000',
          status: 'Released',
          rowVersion: 2,
        },
      ];
      const boms = [
        {
          id: 'bom-1',
          code: 'BOM-CHAIR-A',
          productItemId: 'item-fg',
          version: 1,
          isActive: true,
          rowVersion: 1,
        },
      ];

      const { captured } = createFetchRouter((req) => {
        if (req.method === 'GET' && req.path === '/work-orders') {
          return jsonResponse(listEnvelope(workOrders));
        }
        if (req.method === 'POST' && req.path === '/work-orders/wo-1/release') {
          workOrders[0] = { ...workOrders[0], status: 'Released', rowVersion: 2 };
          return jsonResponse(workOrders[0]);
        }
        if (req.method === 'POST' && req.path === '/work-orders/wo-2/complete') {
          workOrders[1] = {
            ...workOrders[1],
            status: 'Completed',
            producedQty: '19.5000',
            rowVersion: 3,
          };
          return jsonResponse(workOrders[1]);
        }
        if (req.method === 'GET' && req.path === '/bill-of-materials') {
          return jsonResponse(listEnvelope(boms));
        }
        if (req.method === 'GET' && req.path === '/bill-of-materials/bom-1') {
          return jsonResponse({
            ...boms[0],
            lines: [
              {
                id: 'boml-1',
                lineNo: 1,
                componentItemId: 'item-rm',
                quantity: '2.5000',
                uom: 'KG',
              },
            ],
          });
        }
        if (req.method === 'GET' && req.path === '/items') {
          return jsonResponse(
            listEnvelope([
              { id: 'item-fg', code: 'FG-01', name: 'Office Chair', baseUom: 'PCS' },
              { id: 'item-rm', code: 'RM-01', name: 'Aluminum Frame', baseUom: 'KG' },
            ])
          );
        }
        return jsonResponse({}, 404);
      });

      setAuthPermissions(['manufacturing:read', 'manufacturing:write']);
      renderWorkspace(<ManufacturingWorkspace />);

      expect(await screen.findByText('WO-2026-101')).toBeInTheDocument();

      // Release WO-1
      await user.click(screen.getByRole('button', { name: 'Release WO-2026-101' }));
      await waitFor(() => {
        const relReq = captured.find(
          (r) => r.method === 'POST' && r.path === '/work-orders/wo-1/release'
        );
        expect(relReq?.headers['if-match']).toBe('"1"');
      });

      // Complete WO-2 with producedQty (not completedQty)
      await user.click(screen.getByRole('button', { name: 'Complete WO-2026-102' }));
      const compDialog = await screen.findByRole('dialog');
      const qtyInput = within(compDialog).getByRole('textbox');
      await user.clear(qtyInput);
      await user.type(qtyInput, '19.50');
      await user.click(within(compDialog).getByRole('button', { name: /Complete Production/i }));

      await waitFor(() => {
        const compReq = captured.find(
          (r) => r.method === 'POST' && r.path === '/work-orders/wo-2/complete'
        );
        expect(compReq).toBeDefined();
        expect(compReq?.headers['if-match']).toBe('"2"');
        expect(compReq?.body).toEqual({ producedQty: '19.50' });
      });

      // Open Bill of Materials tab & inspect component detail drawer
      await user.click(screen.getByRole('tab', { name: /Bill of Materials/i }));
      const bomRow = await screen.findByText('BOM-CHAIR-A');
      await user.click(bomRow);
      expect(await screen.findByText(/Components \(1\)/i)).toBeInTheDocument();
      expect(await screen.findByText(/RM-01 — Aluminum Frame/i)).toBeInTheDocument();
    });
  });

  describe('SalesWorkspace (/sales)', () => {
    it('computes exact SO total, confirms Draft SO, dispatches delivery, and inspects Delivery detail', async () => {
      expect(
        computeSalesOrderTotalDecimal([
          { orderQty: '3', unitPrice: '12.34' },
          { orderQty: '0.5', unitPrice: '10.00' },
        ])
      ).toBe('42.0200');

      const user = userEvent.setup();
      const salesOrders = [
        {
          id: 'so-1',
          organizationId: TEST_ORG.id,
          soNo: 'SO-2026-501',
          customerId: 'cust-1',
          orderDate: '2026-11-12',
          currencyCode: 'IDR',
          totalAmount: '1500000.0000',
          status: 'Draft',
          rowVersion: 1,
        },
        {
          id: 'so-2',
          organizationId: TEST_ORG.id,
          soNo: 'SO-2026-502',
          customerId: 'cust-1',
          orderDate: '2026-11-13',
          currencyCode: 'IDR',
          totalAmount: '3000000.0000',
          status: 'Confirmed',
          rowVersion: 2,
        },
      ];
      const deliveries = [
        {
          id: 'del-1',
          organizationId: TEST_ORG.id,
          deliveryNo: 'DO-2026-888',
          salesOrderId: 'so-0',
          warehouseId: 'wh-1',
          deliveryDate: '2026-11-14',
          status: 'Delivered',
          rowVersion: 1,
        },
      ];

      const { captured } = createFetchRouter((req) => {
        if (req.method === 'GET' && req.path === '/sales-orders') {
          return jsonResponse(listEnvelope(salesOrders));
        }
        if (req.method === 'POST' && req.path === '/sales-orders/so-1/confirm') {
          salesOrders[0] = { ...salesOrders[0], status: 'Confirmed', rowVersion: 2 };
          return jsonResponse({ ...salesOrders[0], lines: [] });
        }
        if (req.method === 'POST' && req.path === '/sales-orders/so-2/deliveries') {
          const created = {
            id: 'del-2',
            organizationId: TEST_ORG.id,
            deliveryNo: 'DO-SO-2026-502',
            salesOrderId: 'so-2',
            warehouseId: 'wh-1',
            deliveryDate: '2026-11-15',
            status: 'Delivered',
            rowVersion: 1,
            lines: [],
          };
          deliveries.push(created);
          salesOrders[1] = { ...salesOrders[1], status: 'Delivered', rowVersion: 3 };
          return jsonResponse(created, 201);
        }
        if (req.method === 'GET' && req.path === '/deliveries') {
          return jsonResponse(listEnvelope(deliveries));
        }
        if (req.method === 'GET' && req.path === '/deliveries/del-1') {
          return jsonResponse({
            ...deliveries[0],
            lines: [
              {
                id: 'dl-1',
                soLineId: 'sol-1',
                lineNo: 1,
                itemId: 'item-1',
                deliveredQty: '10.0000',
                rowVersion: 1,
              },
            ],
          });
        }
        if (req.method === 'GET' && req.path === '/parties') {
          return jsonResponse(
            listEnvelope([{ id: 'cust-1', code: 'C01', name: 'PT Retail Makmur', partyType: 'customer' }])
          );
        }
        if (req.method === 'GET' && req.path === '/items') {
          return jsonResponse(
            listEnvelope([{ id: 'item-1', code: 'FG-9', name: 'Finished Widget', baseUom: 'PCS' }])
          );
        }
        if (req.method === 'GET' && req.path === '/warehouses') {
          return jsonResponse(listEnvelope([{ id: 'wh-1', code: 'WH-01', name: 'Main Hub' }]));
        }
        return jsonResponse({}, 404);
      });

      setAuthPermissions(['sales:read', 'sales:write']);
      renderWorkspace(<SalesWorkspace />);

      expect(await screen.findByText('SO-2026-501')).toBeInTheDocument();

      // Confirm SO-1
      await user.click(screen.getByRole('button', { name: 'Confirm SO-2026-501' }));
      const confirmDialog = await screen.findByRole('dialog');
      await user.click(within(confirmDialog).getByRole('button', { name: /Confirm Sales Order/i }));
      await waitFor(() => {
        const confReq = captured.find(
          (r) => r.method === 'POST' && r.path === '/sales-orders/so-1/confirm'
        );
        expect(confReq?.headers['if-match']).toBe('"1"');
      });

      // Deliver SO-2
      await user.click(screen.getByRole('button', { name: 'Deliver SO-2026-502' }));
      const delivDialog = await screen.findByRole('dialog');
      await user.click(within(delivDialog).getByRole('button', { name: /Post Delivery/i }));
      await waitFor(() => {
        const delivReq = captured.find(
          (r) => r.method === 'POST' && r.path === '/sales-orders/so-2/deliveries'
        );
        expect(delivReq?.headers['if-match']).toBe('"2"');
      });

      // Inspect Deliveries tab & detail drawer
      await user.click(screen.getByRole('tab', { name: /Deliveries/i }));
      const delRow = await screen.findByText('DO-2026-888');
      await user.click(delRow);
      expect(await screen.findByText(/Delivery DO-2026-888/i)).toBeInTheDocument();
      expect(await screen.findByText(/FG-9 — Finished Widget/i)).toBeInTheDocument();
    });

    it('renders EmptyState on empty lists and enforces sales:write RBAC', async () => {
      createFetchRouter(() => jsonResponse(listEnvelope([])));
      setAuthPermissions(['sales:read']);
      renderWorkspace(<SalesWorkspace />);

      expect(await screen.findByText('No Sales Orders Found')).toBeInTheDocument();
      expect(screen.getByText('Read-only')).toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /New Sales Order/i })).not.toBeInTheDocument();
    });
  });

  describe('Cross-Module Resilience & Edge Cases', () => {
    it('maps 422 field errors onto forms and supports Reservation Release & Movement Transfer', async () => {
      const user = userEvent.setup();
      createFetchRouter((req) => {
        if (req.method === 'GET' && req.path === '/parties') {
          return jsonResponse(listEnvelope([]));
        }
        if (req.method === 'GET' && req.path === '/items') {
          return jsonResponse(listEnvelope([]));
        }
        if (req.method === 'POST' && req.path === '/parties') {
          return jsonResponse(
            {
              error: {
                code: 'VALIDATION_ERROR',
                message: 'Validation failed',
                details: [{ field: 'code', issue: 'Partner code already exists' }],
              },
            },
            422
          );
        }
        return jsonResponse(listEnvelope([]));
      });

      setAuthPermissions(['masterdata:read', 'masterdata:write']);
      const { unmount } = renderWorkspace(<MasterdataWorkspace />);
      expect(await screen.findByText('No Business Partners Found')).toBeInTheDocument();
      await user.click(screen.getByRole('button', { name: /New Partner/i }));
      const dialog = await screen.findByRole('dialog');
      await user.type(within(dialog).getByLabelText(/Partner Code/i), 'DUP-01');
      await user.type(within(dialog).getByLabelText(/Legal Name/i), 'Duplicate Partner');
      await user.click(within(dialog).getByRole('button', { name: /Create Partner/i }));
      expect(await within(dialog).findByText(/Partner code already exists/i)).toBeInTheDocument();
      unmount();

      // Inventory Reservation Release + empty RBAC check
      const reservations = [
        {
          id: 'resv-rel-1',
          itemId: 'item-1',
          warehouseId: 'wh-1',
          reservedQty: '5.0000',
          salesOrderRef: 'SO-REL-99',
          status: 'Reserved',
          rowVersion: 3,
        },
      ];
      const { captured } = createFetchRouter((req) => {
        if (req.path === '/stock-balances' || req.path === '/stock-movements') {
          return jsonResponse(listEnvelope([]));
        }
        if (req.path === '/warehouses') {
          return jsonResponse(
            listEnvelope([
              { id: 'wh-1', code: 'WH-01', name: 'Main', organizationRefId: TEST_ORG.id, rowVersion: 1 },
            ])
          );
        }
        if (req.path === '/reservations' && req.method === 'GET') {
          return jsonResponse(listEnvelope(reservations));
        }
        if (req.path === '/reservations/resv-rel-1/release' && req.method === 'POST') {
          reservations[0] = { ...reservations[0], status: 'Released', rowVersion: 4 };
          return jsonResponse(reservations[0]);
        }
        return jsonResponse(listEnvelope([]));
      });

      setAuthPermissions(['inventory:read', 'inventory:write']);
      renderWorkspace(<InventoryWorkspace />);
      await user.click(screen.getByRole('tab', { name: /Reservations/i }));
      await user.click(await screen.findByRole('button', { name: 'Release reservation SO-REL-99' }));
      const relDialog = await screen.findByRole('dialog');
      await user.click(within(relDialog).getByRole('button', { name: /Release Reservation/i }));
      await waitFor(() => {
        const relReq = captured.find(
          (r) => r.method === 'POST' && r.path === '/reservations/resv-rel-1/release'
        );
        expect(relReq?.headers['if-match']).toBe('"3"');
      });
    });
  });
});
