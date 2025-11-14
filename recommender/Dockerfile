FROM python:3.10-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    git \
    && rm -rf /var/lib/apt/lists/*

# Copy project files
COPY pyproject.toml setup.py requirements.txt ./
COPY src/ ./src/

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Run with uvicorn
CMD ["uvicorn", "boifi_recommender.main:app", "--host", "0.0.0.0", "--port", "8000"]
