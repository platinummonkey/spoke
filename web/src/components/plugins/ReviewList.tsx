import { PluginReview } from '../../types/plugin';
import StarRating from './StarRating';
import './ReviewList.css';

interface ReviewListProps {
  reviews: PluginReview[];
}

export default function ReviewList({ reviews }: ReviewListProps) {
  if (reviews.length === 0) {
    return (
      <div className="empty-reviews">
        <p>No reviews yet. Be the first to review this plugin!</p>
      </div>
    );
  }

  return (
    <div className="review-list">
      {reviews.map((review) => (
        <div key={review.id} className="review-item">
          <div className="review-header">
            <div className="review-author">
              <span className="author-name">{review.user_name || review.user_id}</span>
              <StarRating rating={review.rating} size="small" />
            </div>
            <span className="review-date">
              {new Date(review.created_at).toLocaleDateString()}
            </span>
          </div>

          {review.review && (
            <div className="review-content">
              <p>{review.review}</p>
            </div>
          )}

          {review.updated_at !== review.created_at && (
            <div className="review-footer">
              <span className="edited-label">
                Edited {new Date(review.updated_at).toLocaleDateString()}
              </span>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
