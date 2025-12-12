import { query, mutation } from "./_generated/server";
import { v } from "convex/values";

// List all AI scripts
export const list = query({
  args: {},
  handler: async (ctx) => {
    return await ctx.db.query("aiScripts").collect();
  },
});

// Get a single AI script by ID
export const get = query({
  args: { id: v.id("aiScripts") },
  handler: async (ctx, args) => {
    return await ctx.db.get(args.id);
  },
});

// Get a single AI script by name
export const getByName = query({
  args: { name: v.string() },
  handler: async (ctx, args) => {
    return await ctx.db
      .query("aiScripts")
      .withIndex("by_name", (q) => q.eq("name", args.name))
      .first();
  },
});

// Create a new AI script
export const create = mutation({
  args: {
    name: v.string(),
    code: v.string(),
    description: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    const existing = await ctx.db
      .query("aiScripts")
      .withIndex("by_name", (q) => q.eq("name", args.name))
      .first();
    
    if (existing) {
      throw new Error(`AI script with name "${args.name}" already exists`);
    }

    return await ctx.db.insert("aiScripts", {
      name: args.name,
      code: args.code,
      description: args.description,
      createdAt: Date.now(),
    });
  },
});

// Update an existing AI script
export const update = mutation({
  args: {
    id: v.id("aiScripts"),
    code: v.optional(v.string()),
    description: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    const { id, ...updates } = args;
    const filtered: Record<string, unknown> = { updatedAt: Date.now() };
    
    if (updates.code !== undefined) {
      filtered.code = updates.code;
    }
    if (updates.description !== undefined) {
      filtered.description = updates.description;
    }

    await ctx.db.patch(id, filtered);
    return await ctx.db.get(id);
  },
});

// Delete an AI script
export const remove = mutation({
  args: { id: v.id("aiScripts") },
  handler: async (ctx, args) => {
    await ctx.db.delete(args.id);
  },
});
