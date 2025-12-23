#!/usr/bin/env python
# -*- coding: utf-8 -*-

import sys
import os
import logging
from paddleocr import PaddleOCR

# Add current directory to Python path
current_dir = os.path.dirname(os.path.abspath(__file__))
if current_dir not in sys.path:
    sys.path.append(current_dir)

# Import ImageParser
from parser.image_parser import ImageParser

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)


def init_ocr_model():
    """Initialize PaddleOCR model to pre-download and cache models"""
    try:
        logger.info("Initializing PaddleOCR model for pre-download...")

        # Use the same configuration as in the code
        ocr_config = {
            "use_gpu": False,
            "text_det_limit_type": "max",
            "text_det_limit_side_len": 960,
            "use_doc_orientation_classify": True,  # Enable document orientation classification
            "use_doc_unwarping": False,
            "use_textline_orientation": True,  # Enable text line orientation detection
            "text_recognition_model_name": "PP-OCRv4_server_rec",
            "text_detection_model_name": "PP-OCRv4_server_det",
            "text_det_thresh": 0.3,
            "text_det_box_thresh": 0.6,
            "text_det_unclip_ratio": 1.5,
            "text_rec_score_thresh": 0.0,
            "ocr_version": "PP-OCRv4",
            "lang": "ch",
            "show_log": False,
            "use_dilation": True,
            "det_db_score_mode": "slow",
        }
        
        # Initialize PaddleOCR, this will trigger model download and caching
        ocr = PaddleOCR(**ocr_config)
        logger.info("PaddleOCR model initialization completed successfully")

        # Test OCR functionality to ensure model works properly
        import numpy as np
        from PIL import Image

        # Create a simple test image
        test_image = np.ones((100, 300, 3), dtype=np.uint8) * 255
        test_pil = Image.fromarray(test_image)

        # Execute one OCR test
        result = ocr.ocr(np.array(test_pil), cls=False)
        logger.info("PaddleOCR test completed successfully")
        
    except Exception as e:
        logger.error(f"Failed to initialize PaddleOCR model: {str(e)}")
        raise
