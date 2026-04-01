import {
  liveBucketsCreate,
  liveBucketsDelete,
  liveBucketsExists,
  liveBucketsList,
  liveDelete,
  liveGet,
  liveHead,
  liveKeysList,
  livePut,
  liveStatus,
} from "./objectdb_live.js";
import {
  mockBucketsCreate,
  mockBucketsDelete,
  mockBucketsExists,
  mockBucketsList,
  mockDelete,
  mockGet,
  mockHead,
  mockKeysList,
  mockPut,
  mockStatus,
} from "./objectdb_mock.js";
import { liveConfig } from "./shared.js";

export const helpersObjectDB = {
  async status() {
    const cfg = liveConfig("helpers.ObjectDB");
    return cfg.live ? liveStatus(cfg) : mockStatus();
  },

  buckets: {
    async list() {
      const cfg = liveConfig("helpers.ObjectDB");
      return cfg.live ? liveBucketsList(cfg) : mockBucketsList();
    },

    async create(bucket) {
      const cfg = liveConfig("helpers.ObjectDB");
      return cfg.live ? liveBucketsCreate(cfg, bucket) : mockBucketsCreate(bucket);
    },

    async exists(bucket) {
      const cfg = liveConfig("helpers.ObjectDB");
      return cfg.live ? liveBucketsExists(cfg, bucket) : mockBucketsExists(bucket);
    },

    async delete(bucket) {
      const cfg = liveConfig("helpers.ObjectDB");
      return cfg.live ? liveBucketsDelete(cfg, bucket) : mockBucketsDelete(bucket);
    },
  },

  keys: {
    async list(bucket, options = {}) {
      const cfg = liveConfig("helpers.ObjectDB");
      return cfg.live ? liveKeysList(cfg, bucket, options) : mockKeysList(bucket, options);
    },
  },

  async put(bucket, key, value, options = {}) {
    const cfg = liveConfig("helpers.ObjectDB");
    return cfg.live ? livePut(cfg, bucket, key, value, options) : mockPut(bucket, key, value);
  },

  async get(bucket, key, options = {}) {
    const cfg = liveConfig("helpers.ObjectDB");
    return cfg.live ? liveGet(cfg, bucket, key, options) : mockGet(bucket, key);
  },

  async head(bucket, key) {
    const cfg = liveConfig("helpers.ObjectDB");
    return cfg.live ? liveHead(cfg, bucket, key) : mockHead(bucket, key);
  },

  async delete(bucket, key) {
    const cfg = liveConfig("helpers.ObjectDB");
    return cfg.live ? liveDelete(cfg, bucket, key) : mockDelete(bucket, key);
  },
};
