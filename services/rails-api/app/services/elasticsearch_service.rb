require 'elasticsearch'

class ElasticsearchService
  MESSAGES_INDEX = 'messages'

  def initialize
    @client = Elasticsearch::Client.new(
      url: ENV.fetch('ELASTICSEARCH_URL', 'http://elasticsearch:9200'),
      log: Rails.env.development?
    )
  end

  def search_messages(application_token, chat_number, query, page: 1, per_page: 20)
    from = (page - 1) * per_page

    response = @client.search(
      index: MESSAGES_INDEX,
      body: {
        query: {
          bool: {
            must: [
              { term: { application_token: application_token } },
              { term: { chat_number: chat_number } },
              {
                multi_match: {
                  query: query,
                  fields: ['body', 'body.ngram'],
                  type: 'best_fields',
                  fuzziness: 'AUTO'
                }
              }
            ]
          }
        },
        from: from,
        size: per_page,
        sort: [
          { message_number: { order: 'asc' } }
        ]
      }
    )

    {
      total: response.dig('hits', 'total', 'value') || 0,
      messages: response.dig('hits', 'hits')&.map { |hit| hit['_source'] } || [],
      page: page,
      per_page: per_page
    }
  rescue => e
    Rails.logger.error "Elasticsearch search failed: #{e.message}"
    raise
  end
end
