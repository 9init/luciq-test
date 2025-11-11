class Message < ApplicationRecord
  belongs_to :chat, counter_cache: :messages_count
  validates :number, presence: true, uniqueness: { scope: :chat_id }
  before_validation :set_message_number, on: :create

  private

  def set_message_number
    return if number.present?

    last_number = chat.messages.maximum(:number) || 0
    self.number = last_number + 1
  end
end
