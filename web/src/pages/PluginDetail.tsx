import React, { useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import {
  usePlugin,
  usePluginVersions,
  usePluginReviews,
  useCreateReview,
  useRecordInstallation,
} from '../hooks/usePlugins';
import SecurityBadge from '../components/plugins/SecurityBadge';
import StarRating from '../components/plugins/StarRating';
import ReviewList from '../components/plugins/ReviewList';
import VersionList from '../components/plugins/VersionList';
import './PluginDetail.css';

export default function PluginDetail() {
  const { id } = useParams<{ id: string }>();
  const [activeTab, setActiveTab] = useState<'overview' | 'versions' | 'reviews'>('overview');
  const [showReviewForm, setShowReviewForm] = useState(false);
  const [reviewRating, setReviewRating] = useState(5);
  const [reviewText, setReviewText] = useState('');

  const { data: plugin, isLoading, error } = usePlugin(id!);
  const { data: versions } = usePluginVersions(id!);
  const { data: reviews } = usePluginReviews(id!);
  const createReviewMutation = useCreateReview(id!);
  const recordInstallMutation = useRecordInstallation();

  const handleInstall = () => {
    if (plugin?.latest_version) {
      recordInstallMutation.mutate(
        { pluginId: id!, version: plugin.latest_version },
        {
          onSuccess: () => {
            alert(`Installation of ${plugin.name} ${plugin.latest_version} recorded!`);
          },
        }
      );
    }
  };

  const handleSubmitReview = (e: React.FormEvent) => {
    e.preventDefault();
    createReviewMutation.mutate(
      { rating: reviewRating, review: reviewText },
      {
        onSuccess: () => {
          setShowReviewForm(false);
          setReviewText('');
          setReviewRating(5);
          alert('Review submitted successfully!');
        },
        onError: (error) => {
          alert(`Failed to submit review: ${error instanceof Error ? error.message : 'Unknown error'}`);
        },
      }
    );
  };

  if (isLoading) {
    return (
      <div className="plugin-detail loading">
        <div className="spinner" />
        <p>Loading plugin details...</p>
      </div>
    );
  }

  if (error || !plugin) {
    return (
      <div className="plugin-detail error">
        <h2>Plugin Not Found</h2>
        <p>The requested plugin could not be found.</p>
        <Link to="/plugins" className="back-link">← Back to Marketplace</Link>
      </div>
    );
  }

  return (
    <div className="plugin-detail">
      <div className="plugin-header">
        <Link to="/plugins" className="back-link">← Back to Marketplace</Link>

        <div className="header-content">
          <div className="header-left">
            <h1>{plugin.name}</h1>
            <div className="header-badges">
              <SecurityBadge level={plugin.security_level} />
              <span className="type-badge">{plugin.type}</span>
            </div>
            <p className="plugin-description-full">{plugin.description}</p>
          </div>

          <div className="header-right">
            <button onClick={handleInstall} className="install-button" disabled={!plugin.latest_version}>
              Install {plugin.latest_version || ''}
            </button>
            <div className="plugin-stats-summary">
              <div className="stat">
                <StarRating rating={plugin.avg_rating || 0} size="medium" />
                <span className="stat-label">{plugin.review_count || 0} reviews</span>
              </div>
              <div className="stat">
                <span className="stat-value">{plugin.download_count.toLocaleString()}</span>
                <span className="stat-label">downloads</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="plugin-tabs">
        <button
          className={`tab ${activeTab === 'overview' ? 'active' : ''}`}
          onClick={() => setActiveTab('overview')}
        >
          Overview
        </button>
        <button
          className={`tab ${activeTab === 'versions' ? 'active' : ''}`}
          onClick={() => setActiveTab('versions')}
        >
          Versions ({versions?.length || 0})
        </button>
        <button
          className={`tab ${activeTab === 'reviews' ? 'active' : ''}`}
          onClick={() => setActiveTab('reviews')}
        >
          Reviews ({plugin.review_count || 0})
        </button>
      </div>

      <div className="plugin-content">
        {activeTab === 'overview' && (
          <div className="overview-tab">
            <div className="info-section">
              <h2>Information</h2>
              <dl className="info-list">
                <dt>Author:</dt>
                <dd>{plugin.author}</dd>

                {plugin.license && (
                  <>
                    <dt>License:</dt>
                    <dd>{plugin.license}</dd>
                  </>
                )}

                {plugin.homepage && (
                  <>
                    <dt>Homepage:</dt>
                    <dd>
                      <a href={plugin.homepage} target="_blank" rel="noopener noreferrer">
                        {plugin.homepage}
                      </a>
                    </dd>
                  </>
                )}

                {plugin.repository && (
                  <>
                    <dt>Repository:</dt>
                    <dd>
                      <a href={plugin.repository} target="_blank" rel="noopener noreferrer">
                        {plugin.repository}
                      </a>
                    </dd>
                  </>
                )}

                <dt>Latest Version:</dt>
                <dd>{plugin.latest_version || 'N/A'}</dd>

                <dt>Created:</dt>
                <dd>{new Date(plugin.created_at).toLocaleDateString()}</dd>

                <dt>Updated:</dt>
                <dd>{new Date(plugin.updated_at).toLocaleDateString()}</dd>
              </dl>
            </div>
          </div>
        )}

        {activeTab === 'versions' && (
          <div className="versions-tab">
            <VersionList versions={versions || []} pluginId={id!} />
          </div>
        )}

        {activeTab === 'reviews' && (
          <div className="reviews-tab">
            <div className="reviews-header">
              <h2>User Reviews</h2>
              <button onClick={() => setShowReviewForm(!showReviewForm)} className="write-review-button">
                {showReviewForm ? 'Cancel' : 'Write a Review'}
              </button>
            </div>

            {showReviewForm && (
              <form onSubmit={handleSubmitReview} className="review-form">
                <div className="form-group">
                  <label>Rating:</label>
                  <StarRating
                    rating={reviewRating}
                    size="large"
                    interactive
                    onRatingChange={setReviewRating}
                  />
                </div>

                <div className="form-group">
                  <label htmlFor="review-text">Review:</label>
                  <textarea
                    id="review-text"
                    value={reviewText}
                    onChange={(e) => setReviewText(e.target.value)}
                    rows={5}
                    placeholder="Share your experience with this plugin..."
                    required
                  />
                </div>

                <button type="submit" className="submit-review-button" disabled={createReviewMutation.isPending}>
                  {createReviewMutation.isPending ? 'Submitting...' : 'Submit Review'}
                </button>
              </form>
            )}

            <ReviewList reviews={reviews || []} />
          </div>
        )}
      </div>
    </div>
  );
}
