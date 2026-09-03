#!/usr/bin/env python3
import os
import threading
from typing import List, Optional

import uvicorn
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

try:
    from FlagEmbedding import FlagReranker
except ImportError as exc:
    raise SystemExit(
        "missing dependency: pip install fastapi uvicorn FlagEmbedding"
    ) from exc


class RerankRequest(BaseModel):
    query: str
    documents: Optional[List[str]] = None
    texts: Optional[List[str]] = None
    model: Optional[str] = None


app = FastAPI()
model_name = os.getenv("BGE_RERANK_MODEL", "BAAI/bge-reranker-v2-m3")
reranker = FlagReranker(model_name, use_fp16=os.getenv("BGE_RERANK_FP16", "true").lower() == "true")
reranker_lock = threading.Lock()


@app.get("/health")
def health():
    return {"status": "ok", "model": model_name}


@app.post("/rerank")
def rerank(req: RerankRequest):
    docs = req.documents or req.texts or []
    if not req.query.strip():
        raise HTTPException(status_code=400, detail="query is required")
    if not docs:
        raise HTTPException(status_code=400, detail="documents or texts is required")

    pairs = [[req.query, doc] for doc in docs]
    # Hugging Face fast tokenizers cannot be borrowed concurrently. Serialize
    # access so dependency retries or parallel requests do not crash the worker.
    with reranker_lock:
        scores = reranker.compute_score(pairs, normalize=True)
    if isinstance(scores, float):
        scores = [scores]
    return {"scores": [float(score) for score in scores]}


if __name__ == "__main__":
    host = os.getenv("BGE_RERANK_HOST", "127.0.0.1")
    port = int(os.getenv("BGE_RERANK_PORT", "8000"))
    uvicorn.run(app, host=host, port=port)
