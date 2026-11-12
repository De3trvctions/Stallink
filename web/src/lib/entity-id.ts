const MAX_ENTITY_ID = "9223372036854775807";

export type EntityId = string & { readonly __entityId: unique symbol };

export function parseEntityId(value: unknown): EntityId | null {
  if (typeof value !== "string" || !/^[1-9][0-9]*$/.test(value)) return null;
  if (value.length > MAX_ENTITY_ID.length
    || (value.length === MAX_ENTITY_ID.length && value > MAX_ENTITY_ID)) return null;
  return value as EntityId;
}

export function isEntityId(value: unknown): value is EntityId {
  return parseEntityId(value) !== null;
}

export function hasValidEntityIds(value: unknown): boolean {
  if (Array.isArray(value)) return value.every(hasValidEntityIds);
  if (value === null || typeof value !== "object") return true;

  for (const [key, child] of Object.entries(value)) {
    const isEntityIDField = key === "id" || (key.endsWith("Id") && key !== "clientOrderId");
    if (isEntityIDField && child !== null && !isEntityId(child)) return false;
    if (!hasValidEntityIds(child)) return false;
  }
  return true;
}
