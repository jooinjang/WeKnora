import torch
import uvicorn
from fastapi import FastAPI
from pydantic import BaseModel, Field
from transformers import AutoModelForSequenceClassification, AutoTokenizer
from typing import List

# --- 1. Define API request and response data structures ---

class RerankRequest(BaseModel):
    query: str
    documents: List[str]

class DocumentInfo(BaseModel):
    text: str

class TestRankResult(BaseModel):
    index: int
    document: DocumentInfo
    score: float

class TestFinalResponse(BaseModel):
    results: List[TestRankResult]


# --- 2. Load model (executed once at service startup) ---
print("Loading model, please wait...")
device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
print(f"Using device: {device}")
try:
    model_path = '/data1/home/lwx/work/Download/rerank_model_weight'
    tokenizer = AutoTokenizer.from_pretrained(model_path)
    model = AutoModelForSequenceClassification.from_pretrained(model_path)
    model.to(device)
    model.eval()
    print("Model loaded successfully!")
except Exception as e:
    print(f"Model loading failed: {e}")
    exit()

# --- 3. Create FastAPI application ---
app = FastAPI(
    title="Reranker API (Test Version)",
    description="API service returning 'score' field for Go client compatibility testing",
    version="1.0.1"
)

# --- 4. Define API endpoint ---
@app.post("/rerank", response_model=TestFinalResponse)
def rerank_endpoint(request: RerankRequest):
    pairs = [[request.query, doc] for doc in request.documents]

    with torch.no_grad():
        inputs = tokenizer(pairs, padding=True, truncation=True, return_tensors='pt', max_length=1024).to(device)
        scores = model(**inputs, return_dict=True).logits.view(-1, ).float()

    results = []
    for i, (text, score_val) in enumerate(zip(request.documents, scores)):
        doc_info = DocumentInfo(text=text)
        test_result = TestRankResult(
            index=i,
            document=doc_info,
            score=score_val.item()
        )
        results.append(test_result)

    sorted_results = sorted(results, key=lambda x: x.score, reverse=True)
    return {"results": sorted_results}

@app.get("/")
def read_root():
    return {"status": "Reranker API (Test Version) is running"}

# --- 5. Start service ---
if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
    